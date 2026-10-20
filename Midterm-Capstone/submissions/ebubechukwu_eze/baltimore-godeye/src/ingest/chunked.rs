//! Parallel CSV ingestion: one reader thread, N parser threads.
//!
//! ```text
//!   file --read 4 MiB--> [reader] --bounded channel--> [parser] x N --> Vec<Record>
//!                            ^                             |
//!                            +------ buffer pool ----------+
//! ```
//!
//! * The reader cuts the file into chunks that end on a record boundary
//!   (quote-aware, so a newline inside a quoted field never splits a row).
//! * The chunk channel is bounded: if the parsers fall behind, the reader
//!   blocks instead of buffering the file, so peak memory is a fixed number of
//!   chunks no matter how large the input is.
//! * Parsed-out chunk buffers flow back through a pool channel and are reused,
//!   so steady-state ingestion allocates no new I/O buffers.
//! * Parsers are generic over `CsvSchema`, so each dataset gets its own
//!   monomorphised loop with no dynamic dispatch per row.

use std::fs::File;
use std::io::Read;
use std::path::Path;
use std::thread;
use std::time::Instant;

use crossbeam_channel::{bounded, unbounded};

use crate::error::AppError;
use crate::model::{IngestStats, Interner};

const CHUNK_BYTES: usize = 4 << 20;

/// Shared vocabularies the parsers intern into.
pub struct ParseCtx<'a> {
    pub neighborhoods: &'a Interner,
    pub blocks: &'a Interner,
    pub sr_types: &'a Interner,
}

/// Per-thread reusable string buffers for building join keys.
#[derive(Default)]
pub struct Scratch {
    pub key: String,
}

/// One CSV row, exposing only the columns the schema asked for.
pub struct Row<'a> {
    record: &'a csv::ByteRecord,
    columns: &'a [usize],
}

impl<'a> Row<'a> {
    /// The i-th column of `CsvSchema::COLUMNS`, borrowed straight from the
    /// chunk buffer. Missing or non-UTF-8 fields read as "".
    pub fn get(&self, i: usize) -> &'a str {
        self.record
            .get(self.columns[i])
            .and_then(|bytes| std::str::from_utf8(bytes).ok())
            .map_or("", str::trim)
    }
}

/// Describes how one dataset's rows become typed records.
pub trait CsvSchema {
    type Record: Send;
    const DATASET: &'static str;
    /// Columns looked up by header name, so column order in the file is free to change.
    const COLUMNS: &'static [&'static str];
    /// Returns `None` to reject a row that cannot be used.
    fn parse(row: &Row<'_>, ctx: &ParseCtx<'_>, scratch: &mut Scratch) -> Option<Self::Record>;
}

/// Index just past the last newline that is outside a quoted field.
/// `buf` must start at a record boundary.
pub fn last_record_boundary(buf: &[u8]) -> Option<usize> {
    let mut in_quotes = false;
    let mut boundary = None;
    for (i, &b) in buf.iter().enumerate() {
        match b {
            b'"' => in_quotes = !in_quotes,
            b'\n' if !in_quotes => boundary = Some(i + 1),
            _ => {}
        }
    }
    boundary
}

fn resolve_columns<S: CsvSchema>(header: &[u8]) -> Result<Vec<usize>, AppError> {
    let header = header.strip_prefix(b"\xEF\xBB\xBF").unwrap_or(header);
    let mut reader = csv::ReaderBuilder::new().has_headers(false).from_reader(header);
    let mut record = csv::ByteRecord::new();
    let _ = reader.read_byte_record(&mut record);
    S::COLUMNS
        .iter()
        .map(|&wanted| {
            record
                .iter()
                .position(|name| name.trim_ascii().eq_ignore_ascii_case(wanted.as_bytes()))
                .ok_or(AppError::MissingColumn { dataset: S::DATASET, column: wanted })
        })
        .collect()
}

fn read_block(file: &mut File, buf: &mut Vec<u8>, path: &Path) -> Result<usize, AppError> {
    file.take(CHUNK_BYTES as u64).read_to_end(buf).map_err(|e| AppError::io(path, e))
}

pub fn ingest_csv<S: CsvSchema>(
    path: &Path,
    ctx: &ParseCtx<'_>,
    parser_threads: usize,
) -> Result<(Vec<S::Record>, IngestStats), AppError> {
    let started = Instant::now();
    let parser_threads = parser_threads.max(1);
    let mut file = File::open(path).map_err(|e| AppError::io(path, e))?;
    let bytes = file.metadata().map_err(|e| AppError::io(path, e))?.len();

    // The header is resolved before any parser starts, so every parser gets
    // the same immutable column map by shared reference.
    let mut carry = Vec::with_capacity(CHUNK_BYTES);
    read_block(&mut file, &mut carry, path)?;
    let header_end = carry.iter().position(|&b| b == b'\n').map_or(carry.len(), |i| i + 1);
    let columns = resolve_columns::<S>(&carry[..header_end])?;
    carry.drain(..header_end);
    let columns = &columns[..];

    let (chunk_tx, chunk_rx) = bounded::<Vec<u8>>(parser_threads * 2);
    let (pool_tx, pool_rx) = unbounded::<Vec<u8>>();

    let (parts, read_result) = thread::scope(|scope| {
        let parsers: Vec<_> = (0..parser_threads)
            .map(|_| {
                let (chunks, pool) = (chunk_rx.clone(), pool_tx.clone());
                scope.spawn(move || {
                    let mut out: Vec<S::Record> = Vec::new();
                    let mut record = csv::ByteRecord::new();
                    let mut scratch = Scratch::default();
                    let mut rows = 0u64;
                    for chunk in chunks.iter() {
                        let mut reader = csv::ReaderBuilder::new()
                            .has_headers(false)
                            .flexible(true)
                            .from_reader(&chunk[..]);
                        while let Ok(true) = reader.read_byte_record(&mut record) {
                            rows += 1;
                            let row = Row { record: &record, columns };
                            if let Some(parsed) = S::parse(&row, ctx, &mut scratch) {
                                out.push(parsed);
                            }
                        }
                        // Hand the buffer back; if the reader is gone it is simply freed.
                        let _ = pool.send(chunk);
                    }
                    (out, rows)
                })
            })
            .collect();
        drop(chunk_rx);
        drop(pool_tx);

        let read_result = (|| -> Result<(), AppError> {
            loop {
                let mut buf = pool_rx.try_recv().unwrap_or_else(|_| Vec::with_capacity(CHUNK_BYTES * 2));
                buf.clear();
                buf.append(&mut carry);
                let read = read_block(&mut file, &mut buf, path)?;
                if read == 0 {
                    if !buf.is_empty() {
                        let _ = chunk_tx.send(buf);
                    }
                    return Ok(());
                }
                match last_record_boundary(&buf) {
                    Some(end) => {
                        carry.extend_from_slice(&buf[end..]);
                        buf.truncate(end);
                        if chunk_tx.send(buf).is_err() {
                            return Err(AppError::WorkerPanic("CSV parsing"));
                        }
                    }
                    // A single record longer than the chunk: keep reading.
                    None => carry = buf,
                }
            }
        })();
        drop(chunk_tx); // closes the channel so parsers drain and exit

        let parts: Vec<_> = parsers.into_iter().map(|p| p.join()).collect();
        (parts, read_result)
    });
    read_result?;

    let mut rows = 0;
    let mut merged = Vec::new();
    let mut outputs = Vec::with_capacity(parts.len());
    for part in parts {
        let (records, part_rows) = part.map_err(|_| AppError::WorkerPanic("CSV parsing"))?;
        rows += part_rows;
        outputs.push(records);
    }
    merged.reserve_exact(outputs.iter().map(Vec::len).sum());
    for mut records in outputs {
        merged.append(&mut records);
    }

    let stats = IngestStats {
        dataset: S::DATASET,
        file: path.file_name().map(|n| n.to_string_lossy().into_owned()).unwrap_or_default(),
        format: "CSV",
        bytes,
        rows,
        kept: merged.len() as u64,
        parser_threads,
        seconds: started.elapsed().as_secs_f64(),
    };
    Ok((merged, stats))
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Pairs;
    impl CsvSchema for Pairs {
        type Record = (u32, String);
        const DATASET: &'static str = "pairs";
        const COLUMNS: &'static [&'static str] = &["id", "note"];
        fn parse(row: &Row<'_>, _: &ParseCtx<'_>, _: &mut Scratch) -> Option<Self::Record> {
            Some((row.get(0).parse().ok()?, row.get(1).to_string()))
        }
    }

    #[test]
    fn boundary_ignores_newlines_inside_quotes() {
        assert_eq!(last_record_boundary(b"a,b\nc,d\ne"), Some(8));
        assert_eq!(last_record_boundary(b"a,\"x\ny\"\nb,\"open\nstill"), Some(8));
        assert_eq!(last_record_boundary(b"a,\"he said \"\"hi\"\"\n"), None);
        assert_eq!(last_record_boundary(b"no newline"), None);
    }

    #[test]
    fn parallel_ingest_matches_row_count_and_content() {
        // Big enough to span several chunks, with quoted commas and newlines,
        // a BOM, a reordered header and some unparseable rows.
        let path = std::env::temp_dir().join(format!("bi-chunk-test-{}.csv", std::process::id()));
        let mut text = String::from("\u{feff}note,extra,id\r\n");
        let rows = 300_000u32;
        for i in 0..rows {
            match i % 1000 {
                0 => text.push_str(&format!("\"multi\nline, {i}\",x,{i}\r\n")),
                1 => text.push_str("bad row,x,not-a-number\r\n"),
                _ => text.push_str(&format!("note {i},padding padding padding padding,{i}\r\n")),
            }
        }
        std::fs::write(&path, &text).unwrap();
        assert!(text.len() > CHUNK_BYTES * 2);

        let (n, b, t) = (Interner::new(), Interner::new(), Interner::new());
        let ctx = ParseCtx { neighborhoods: &n, blocks: &b, sr_types: &t };
        let (mut records, stats) = ingest_csv::<Pairs>(&path, &ctx, 4).unwrap();
        std::fs::remove_file(&path).unwrap();

        assert_eq!(stats.rows, u64::from(rows));
        assert_eq!(records.len() as u32, rows - rows / 1000);
        records.sort();
        assert_eq!(records[0], (0, "multi\nline, 0".to_string()));
        assert_eq!(records[1], (2, "note 2".to_string()));
        let sum: u64 = records.iter().map(|r| u64::from(r.0)).sum();
        let skipped: u64 = (0..rows).filter(|i| i % 1000 == 1).map(u64::from).sum();
        assert_eq!(sum, u64::from(rows) * u64::from(rows - 1) / 2 - skipped);
    }

    #[test]
    fn missing_column_is_reported() {
        let path = std::env::temp_dir().join(format!("bi-header-test-{}.csv", std::process::id()));
        std::fs::write(&path, "id,other\n1,2\n").unwrap();
        let (n, b, t) = (Interner::new(), Interner::new(), Interner::new());
        let ctx = ParseCtx { neighborhoods: &n, blocks: &b, sr_types: &t };
        let err = ingest_csv::<Pairs>(&path, &ctx, 2).unwrap_err();
        std::fs::remove_file(&path).unwrap();
        assert!(matches!(err, AppError::MissingColumn { column: "note", .. }));
    }
}

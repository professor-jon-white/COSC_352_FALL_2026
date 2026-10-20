//! baltimore-godeye: ingest five Open Baltimore datasets concurrently, join
//! them by neighborhood, street block and distance, and answer five questions
//! for city stakeholders. See README.md for the architecture.

mod analysis;
mod cli;
mod error;
mod geo;
mod ingest;
mod matching;
mod model;
mod normalize;
mod pipeline;
mod report;
mod serve;
mod stats;

use std::fs;
use std::io::{IsTerminal, Write};
use std::process::ExitCode;

use error::AppError;
use report::{renderer, Format};

fn run() -> Result<(), AppError> {
    let options = cli::parse(std::env::args().skip(1))?;
    if options.help {
        print!("{}", cli::USAGE);
        return Ok(());
    }
    let pipeline = options.builder.build()?;
    let report = pipeline.run(&|message| eprintln!("{message}"))?;

    fs::create_dir_all(&options.out_dir).map_err(|e| AppError::io(&options.out_dir, e))?;
    let mut html = String::new();
    for format in [Format::Html, Format::Json, Format::Terminal { color: false }] {
        let renderer = renderer(format);
        let path = options.out_dir.join(format!("report.{}", renderer.extension()));
        let text = renderer.render(&report);
        fs::write(&path, &text).map_err(|e| AppError::io(&path, e))?;
        eprintln!("wrote {}", path.display());
        if format == Format::Html {
            html = text;
        }
    }

    if !options.quiet {
        let color = options.color.unwrap_or_else(|| std::io::stdout().is_terminal());
        let text = renderer(Format::Terminal { color }).render(&report);
        // A closed pipe (e.g. `| head`) is a normal way to stop reading, not an error.
        let _ = std::io::stdout().lock().write_all(text.as_bytes());
    }
    if let Some(port) = options.serve {
        serve::serve(html, port)?;
    }
    Ok(())
}

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            eprintln!("error: {e}");
            ExitCode::FAILURE
        }
    }
}

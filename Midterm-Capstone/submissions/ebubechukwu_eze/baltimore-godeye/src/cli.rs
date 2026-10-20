//! Command-line parsing (hand-rolled: five flags do not justify a dependency).

use std::path::PathBuf;

use crate::error::AppError;
use crate::pipeline::{Pipeline, PipelineBuilder};

pub const USAGE: &str = "\
baltimore-godeye: cross-dataset analytics over Open Baltimore data

USAGE:
    baltimore-godeye [OPTIONS]

OPTIONS:
    --data-dir <DIR>        Directory holding the raw datasets          [default: data]
    --out-dir <DIR>         Where report.html / report.json / report.txt go [default: output]
    --threads <N>           Worker threads                              [default: all cores]
    --camera-radius <M>     Camera coverage radius in metres            [default: 150]
    --top <N>               Rows in each ranking table                  [default: 10]
    --question <ID>         Run only q1..q5 (repeatable)                [default: all]
    --serve [PORT]          After the run, serve the HTML report on localhost [default port: 8080]
    --no-color              Plain terminal output
    --quiet                 Do not print the report to the terminal (files are still written)
    -h, --help              Show this help
";

pub struct Options {
    pub builder: PipelineBuilder,
    pub out_dir: PathBuf,
    pub color: Option<bool>,
    pub quiet: bool,
    pub serve: Option<u16>,
    pub help: bool,
}

fn number<T: std::str::FromStr>(flag: &str, value: Option<String>) -> Result<T, AppError> {
    let value = value.ok_or_else(|| AppError::Config(format!("{flag} needs a value")))?;
    value.parse().map_err(|_| AppError::Config(format!("{flag}: '{value}' is not a valid number")))
}

pub fn parse(args: impl Iterator<Item = String>) -> Result<Options, AppError> {
    let mut options = Options {
        builder: Pipeline::builder(),
        out_dir: PathBuf::from("output"),
        color: None,
        quiet: false,
        serve: None,
        help: false,
    };
    let mut args = args.peekable();
    while let Some(arg) = args.next() {
        let needs_value = |v: Option<String>| v.ok_or_else(|| AppError::Config(format!("{arg} needs a value")));
        match arg.as_str() {
            "-h" | "--help" => options.help = true,
            "--data-dir" => options.builder = options.builder.data_dir(needs_value(args.next())?),
            "--out-dir" => options.out_dir = PathBuf::from(needs_value(args.next())?),
            "--threads" => options.builder = options.builder.threads(number(&arg, args.next())?),
            "--camera-radius" => options.builder = options.builder.camera_radius_m(number(&arg, args.next())?),
            "--top" => options.builder = options.builder.top_n(number(&arg, args.next())?),
            "--question" => options.builder = options.builder.question(needs_value(args.next())?),
            "--no-color" => options.color = Some(false),
            "--quiet" => options.quiet = true,
            "--serve" => {
                let port = args.next_if(|next| !next.starts_with('-'));
                options.serve = Some(match port {
                    Some(p) => number("--serve", Some(p))?,
                    None => 8080,
                });
            }
            other => return Err(AppError::Config(format!("unknown option '{other}' (try --help)"))),
        }
    }
    Ok(options)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse_str(line: &str) -> Result<Options, AppError> {
        parse(line.split_whitespace().map(String::from))
    }

    #[test]
    fn parses_flags() {
        let o = parse_str("--threads 2 --serve --quiet --out-dir x").unwrap();
        assert_eq!((o.serve, o.quiet, o.out_dir), (Some(8080), true, PathBuf::from("x")));
        assert_eq!(parse_str("--serve 9000").unwrap().serve, Some(9000));
        assert_eq!(parse_str("--no-color").unwrap().color, Some(false));
    }

    #[test]
    fn rejects_bad_input() {
        assert!(parse_str("--threads many").is_err());
        assert!(parse_str("--threads").is_err());
        assert!(parse_str("--frobnicate").is_err());
        assert!(parse_str("--threads 0").unwrap().builder.build().is_err());
    }
}

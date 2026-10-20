//! One error type for the whole application, so every stage can use `?`.

use std::fmt;
use std::io;
use std::path::{Path, PathBuf};

#[derive(Debug)]
pub enum AppError {
    Io { path: PathBuf, source: io::Error },
    Json { path: PathBuf, source: serde_json::Error },
    MissingColumn { dataset: &'static str, column: &'static str },
    Malformed { path: PathBuf, reason: String },
    Config(String),
    WorkerPanic(&'static str),
}

impl AppError {
    pub fn io(path: &Path, source: io::Error) -> Self {
        AppError::Io { path: path.to_path_buf(), source }
    }
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::Io { path, source } => {
                write!(f, "{}: {source}", path.display())?;
                if source.kind() == io::ErrorKind::NotFound {
                    write!(f, " (run scripts/fetch_data.sh to download the datasets)")?;
                }
                Ok(())
            }
            AppError::Json { path, source } => write!(f, "{}: invalid JSON: {source}", path.display()),
            AppError::MissingColumn { dataset, column } => {
                write!(f, "dataset '{dataset}' has no column named '{column}'")
            }
            AppError::Malformed { path, reason } => write!(f, "{}: {reason}", path.display()),
            AppError::Config(msg) => write!(f, "{msg}"),
            AppError::WorkerPanic(stage) => write!(f, "a worker thread panicked during {stage}"),
        }
    }
}

impl std::error::Error for AppError {}

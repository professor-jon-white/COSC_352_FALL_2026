//! A deliberately tiny HTTP server for viewing the HTML report (`--serve`).
//! It binds to localhost only and serves one in-memory page.

use std::io::{BufRead, BufReader, Write};
use std::net::TcpListener;
use std::sync::Arc;
use std::thread;

use crate::error::AppError;

pub fn serve(html: String, port: u16) -> Result<(), AppError> {
    let listener = TcpListener::bind(("127.0.0.1", port))
        .map_err(|e| AppError::Config(format!("cannot listen on 127.0.0.1:{port}: {e}")))?;
    println!("Serving the report at http://127.0.0.1:{port}/  (Ctrl-C to stop)");
    let html = Arc::new(html);
    for stream in listener.incoming() {
        let Ok(mut stream) = stream else { continue };
        let html = Arc::clone(&html);
        // One short-lived thread per connection; the page is shared, not copied.
        thread::spawn(move || {
            let mut request_line = String::new();
            if BufReader::new(&stream).read_line(&mut request_line).is_err() {
                return;
            }
            let path = request_line.split_whitespace().nth(1).unwrap_or("/");
            let (status, body) = if path == "/" || path.starts_with("/?") || path == "/index.html" {
                ("200 OK", html.as_str())
            } else {
                ("404 Not Found", "<h1>404</h1>")
            };
            let _ = write!(
                stream,
                "HTTP/1.1 {status}\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                body.len()
            );
        });
    }
    Ok(())
}

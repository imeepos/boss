#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{WebviewUrl, WebviewWindowBuilder};

const WIN_TITLE: &str = "BOSS 管理端";
const WIN_WIDTH: f64 = 1440.0;
const WIN_HEIGHT: f64 = 900.0;

fn main() -> tauri::Result<()> {
    tauri::Builder::default()
        .setup(|app| {
            WebviewWindowBuilder::new(app, "main", WebviewUrl::default())
                .title(WIN_TITLE)
                .inner_size(WIN_WIDTH, WIN_HEIGHT)
                .min_inner_size(1100.0, 700.0)
                .build()?;
            Ok(())
        })
        .run(tauri::generate_context!())
}

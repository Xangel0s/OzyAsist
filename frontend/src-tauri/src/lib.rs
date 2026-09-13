use tauri::menu::{Menu, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{Emitter, Manager};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, ShortcutState};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, shortcut, event| {
                    if event.state() == ShortcutState::Pressed {
                        let shortcut_str = shortcut.to_string();
                        // 1. Atajo Principal: Alt + Space (Toggle Visibilidad / Foco)
                        if (shortcut_str.contains("Alt") || shortcut_str.contains("Option")) && shortcut_str.contains("Space") {
                            if let Some(window) = app.get_webview_window("main") {
                                if window.is_visible().unwrap_or(false) {
                                    let _ = window.hide();
                                } else {
                                    let _ = window.show();
                                    let _ = window.set_focus();
                                }
                            }
                        }
                        // 2. Kill-Switch de Pánico: Ctrl + Shift + Alt + K
                        if (shortcut_str.contains("Control") || shortcut_str.contains("Ctrl"))
                            && shortcut_str.contains("Shift")
                            && (shortcut_str.contains("Alt") || shortcut_str.contains("Option"))
                            && (shortcut_str.contains("KeyK") || shortcut_str.contains("K"))
                        {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.emit("ozy:panic_kill", ());
                            }
                        }
                    }
                })
                .build(),
        )
        .setup(|app| {
            #[cfg(desktop)]
            {
                let _ = app.global_shortcut().register("Alt+Space");
                let _ = app.global_shortcut().register("Ctrl+Shift+Alt+K");

                // Configuración de Bandeja del Sistema (System Tray)
                let toggle_i = MenuItem::with_id(app, "toggle", "Mostrar / Ocultar OzyAssist", true, None::<&str>)?;
                let kill_i = MenuItem::with_id(app, "kill", "Freno de Pánico (Kill-Switch)", true, None::<&str>)?;
                let quit_i = MenuItem::with_id(app, "quit", "Salir de OzyAssist", true, None::<&str>)?;
                let menu = Menu::with_items(app, &[&toggle_i, &kill_i, &quit_i])?;

                let _ = TrayIconBuilder::new()
                    .icon(app.default_window_icon().unwrap().clone())
                    .menu(&menu)
                    .show_menu_on_left_click(false)
                    .on_menu_event(|app, event| match event.id.as_ref() {
                        "toggle" => {
                            if let Some(window) = app.get_webview_window("main") {
                                if window.is_visible().unwrap_or(false) {
                                    let _ = window.hide();
                                } else {
                                    let _ = window.show();
                                    let _ = window.set_focus();
                                }
                            }
                        }
                        "kill" => {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.emit("ozy:panic_kill", ());
                            }
                        }
                        "quit" => {
                            app.exit(0);
                        }
                        _ => {}
                    })
                    .on_tray_icon_event(|tray, event| {
                        if let TrayIconEvent::Click {
                            button: MouseButton::Left,
                            button_state: MouseButtonState::Up,
                            ..
                        } = event
                        {
                            let app = tray.app_handle();
                            if let Some(window) = app.get_webview_window("main") {
                                if window.is_visible().unwrap_or(false) {
                                    let _ = window.hide();
                                } else {
                                    let _ = window.show();
                                    let _ = window.set_focus();
                                }
                            }
                        }
                    })
                    .build(app);
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                // Al cerrar la ventana, ocultarla en la bandeja del sistema en lugar de cerrar el proceso
                let _ = window.hide();
                api.prevent_close();
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

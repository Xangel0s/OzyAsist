use nokhwa::pixel_format::RgbFormat;
use nokhwa::utils::{ApiBackend, CameraIndex, RequestedFormat, RequestedFormatType};
use nokhwa::Camera;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::env;
use std::fs;
use std::io::{self, BufRead};
use std::path::Path;
use windows::{
    core::{w, PCWSTR},
    Win32::Foundation::{BOOL, HWND, LPARAM},
    Win32::UI::Shell::ShellExecuteW,
    Win32::UI::WindowsAndMessaging::{
        AllowSetForegroundWindow, EnumChildWindows, GetClassNameW, GetForegroundWindow,
        GetWindowTextW, IsWindowVisible, ASFW_ANY, SW_SHOWNORMAL,
    },
};

#[derive(Deserialize, Debug)]
struct RpcRequest {
    #[allow(dead_code)]
    jsonrpc: String,
    id: Option<Value>,
    method: String,
    params: Option<Value>,
}

#[derive(Serialize, Debug)]
struct RpcResponse {
    jsonrpc: String,
    id: Value,
    result: Option<Value>,
    error: Option<RpcError>,
}

#[derive(Serialize, Debug)]
struct RpcError {
    code: i32,
    message: String,
}

#[derive(Serialize, Debug)]
struct ToolListResult {
    tools: Vec<Tool>,
}

#[derive(Serialize, Debug)]
struct Tool {
    name: String,
    description: String,
    #[serde(rename = "inputSchema")]
    input_schema: Value,
}

#[derive(Serialize, Debug)]
struct ToolCallResult {
    content: Vec<ToolContent>,
    #[serde(rename = "isError")]
    is_error: Option<bool>,
}

#[derive(Serialize, Debug)]
struct ToolContent {
    #[serde(rename = "type")]
    content_type: String,
    text: String,
}

struct ControlItem {
    class_name: String,
    text: String,
}

unsafe extern "system" fn enum_children_callback(hwnd: HWND, lparam: LPARAM) -> BOOL {
    unsafe {
        if !IsWindowVisible(hwnd).as_bool() {
            return BOOL(1);
        }

        let mut class_buf = [0u16; 256];
        let class_len = GetClassNameW(hwnd, &mut class_buf);
        let class_name = String::from_utf16_lossy(&class_buf[..class_len as usize]);

        let mut text_buf = [0u16; 512];
        let text_len = GetWindowTextW(hwnd, &mut text_buf);
        let text = String::from_utf16_lossy(&text_buf[..text_len as usize]);

        let trimmed_text = text.trim();
        if !trimmed_text.is_empty()
            || class_name.contains("Button")
            || class_name.contains("Edit")
            || class_name.contains("Tab")
            || class_name.contains("Static")
        {
            let controls = &mut *(lparam.0 as *mut Vec<ControlItem>);
            if controls.len() < 30 {
                controls.push(ControlItem {
                    class_name,
                    text: trimmed_text.to_string(),
                });
            }
        }

        BOOL(1)
    }
}

fn main() {
    let args: Vec<String> = env::args().collect();

    // Si se invoca con argumentos CLI directos, ejecutarlos y salir
    if args.len() > 1 {
        match args[1].as_str() {
            "camera-capture" => {
                let mut output = "camera_snapshot.jpg".to_string();
                let mut device: u32 = 0;
                let mut i = 2;
                while i < args.len() {
                    if args[i] == "--output" && i + 1 < args.len() {
                        output = args[i + 1].clone();
                        i += 2;
                    } else if args[i] == "--device" && i + 1 < args.len() {
                        device = args[i + 1].parse().unwrap_or(0);
                        i += 2;
                    } else {
                        i += 1;
                    }
                }
                match capture_camera_frame(&output, device) {
                    Ok(msg) => {
                        println!("{}", msg);
                        std::process::exit(0);
                    }
                    Err(e) => {
                        eprintln!("ERROR: {}", e);
                        std::process::exit(1);
                    }
                }
            }
            "camera-list" => match list_cameras() {
                Ok(cams) => {
                    println!("{}", cams);
                    std::process::exit(0);
                }
                Err(e) => {
                    eprintln!("ERROR: {}", e);
                    std::process::exit(1);
                }
            },
            "skeletonize" => {
                if args.len() > 2 {
                    match skeletonize_file(&args[2]) {
                        Ok(res) => {
                            println!("{}", res);
                            std::process::exit(0);
                        }
                        Err(e) => {
                            eprintln!("ERROR: {}", e);
                            std::process::exit(1);
                        }
                    }
                } else {
                    eprintln!("ERROR: Uso: ozy-core skeletonize <file_path>");
                    std::process::exit(1);
                }
            }
            "inspect-ui" => match inspect_active_ui() {
                Ok(res) => {
                    println!("{}", res);
                    std::process::exit(0);
                }
                Err(e) => {
                    eprintln!("ERROR: {}", e);
                    std::process::exit(1);
                }
            },
            "launch" => {
                if args.len() > 2 {
                    match launch_app(&args[2]) {
                        Ok(_) => {
                            println!("Lanzado exitosamente: {}", args[2]);
                            std::process::exit(0);
                        }
                        Err(e) => {
                            eprintln!("ERROR: {}", e);
                            std::process::exit(1);
                        }
                    }
                } else {
                    eprintln!("ERROR: Uso: ozy-core launch <target>");
                    std::process::exit(1);
                }
            }
            "--version" | "-v" => {
                println!("ozy-core v0.3.0 (Windows Native Engine en Rust + UI Automation + Skeletonizer)");
                std::process::exit(0);
            }
            _ => {
                // Modo desconocido en CLI, continuar a loop JSON-RPC
            }
        }
    }

    // Modo Servidor JSON-RPC / MCP por stdin/stdout
    run_jsonrpc_loop();
}

fn run_jsonrpc_loop() {
    let stdin = io::stdin();
    let handle = stdin.lock();

    for line in handle.lines() {
        let Ok(line) = line else { break };
        if line.trim().is_empty() {
            continue;
        }

        let Ok(req) = serde_json::from_str::<RpcRequest>(&line) else {
            send_error(Value::Null, -32700, "Parse error");
            continue;
        };

        let req_id = req.id.unwrap_or(Value::Null);

        match req.method.as_str() {
            "initialize" => {
                let res = serde_json::json!({
                    "protocolVersion": "2024-11-05",
                    "serverInfo": {
                        "name": "ozy-core",
                        "version": "0.3.0"
                    },
                    "capabilities": {
                        "tools": {}
                    }
                });
                send_result(req_id, res);
            }
            "notifications/initialized" => {
                // No-op
            }
            "tools/list" => {
                let tools = vec![
                    Tool {
                        name: "os_launch_app".to_string(),
                        description: "Ejecuta o abre una aplicación, URL, panel de Windows, o redacta correos usando mailto: (Nativo Rust Win32).".to_string(),
                        input_schema: serde_json::json!({
                            "type": "object",
                            "properties": {
                                "target": {
                                    "type": "string",
                                    "description": "Nombre del ejecutable, ruta, URL, ms-settings:, o mailto: para redactar correos."
                                }
                            },
                            "required": ["target"]
                        }),
                    },
                    Tool {
                        name: "os_camera_capture".to_string(),
                        description: "Captura una fotografía en vivo desde la cámara web de Windows usando el motor nativo MediaFoundation en Rust.".to_string(),
                        input_schema: serde_json::json!({
                            "type": "object",
                            "properties": {
                                "outputPath": {
                                    "type": "string",
                                    "description": "Ruta de archivo destino donde se guardará la imagen (ej: ~/.ozy/workspace/camera.jpg)"
                                },
                                "deviceIndex": {
                                    "type": "integer",
                                    "description": "Índice de la cámara (por defecto 0)"
                                }
                            }
                        }),
                    },
                    Tool {
                        name: "os_camera_list".to_string(),
                        description: "Enumera las cámaras web físicas conectadas al equipo con sus índices y nombres.".to_string(),
                        input_schema: serde_json::json!({
                            "type": "object",
                            "properties": {}
                        }),
                    },
                    Tool {
                        name: "os_skeletonize".to_string(),
                        description: "Extrae el esqueleto estructural y firmas de un archivo de código fuente (.go, .rs, .ts, .py, etc.) ahorrando hasta un 95% de tokens de contexto.".to_string(),
                        input_schema: serde_json::json!({
                            "type": "object",
                            "properties": {
                                "path": {
                                    "type": "string",
                                    "description": "Ruta absoluta o relativa del archivo de código a esqueletonizar"
                                }
                            },
                            "required": ["path"]
                        }),
                    },
                    Tool {
                        name: "os_inspect_active_ui".to_string(),
                        description: "Inspecciona el árbol de accesibilidad Win32 de la ventana activa en pantalla en 1 ms, extrayendo botones, campos de texto y diálogos con CERO tokens de visión multimodal.".to_string(),
                        input_schema: serde_json::json!({
                            "type": "object",
                            "properties": {}
                        }),
                    },
                ];
                let result = ToolListResult { tools };
                send_result(req_id, serde_json::to_value(result).unwrap());
            }
            "tools/call" => {
                if let Some(params) = req.params {
                    let name = params["name"].as_str().unwrap_or("");
                    let arguments = params["arguments"].as_object().cloned().unwrap_or_default();

                    match name {
                        "os_launch_app" => {
                            if let Some(target) = arguments.get("target").and_then(|v| v.as_str()) {
                                match launch_app(target) {
                                    Ok(_) => send_tool_success(req_id, format!("Lanzado exitosamente (Nativo Rust): {}", target)),
                                    Err(e) => send_tool_error(req_id, format!("Error al lanzar: {}", e)),
                                }
                            } else {
                                send_error(req_id, -32602, "Missing 'target' parameter");
                            }
                        }
                        "os_camera_capture" => {
                            let out_path = arguments
                                .get("outputPath")
                                .and_then(|v| v.as_str())
                                .unwrap_or("camera_capture.jpg");
                            let dev_idx = arguments
                                .get("deviceIndex")
                                .and_then(|v| v.as_u64())
                                .unwrap_or(0) as u32;

                            match capture_camera_frame(out_path, dev_idx) {
                                Ok(msg) => send_tool_success(req_id, msg),
                                Err(e) => send_tool_error(req_id, format!("Error en captura de cámara: {}", e)),
                            }
                        }
                        "os_camera_list" => {
                            match list_cameras() {
                                Ok(cams) => send_tool_success(req_id, cams),
                                Err(e) => send_tool_error(req_id, format!("Error listando cámaras: {}", e)),
                            }
                        }
                        "os_skeletonize" => {
                            if let Some(path) = arguments.get("path").and_then(|v| v.as_str()) {
                                match skeletonize_file(path) {
                                    Ok(res) => send_tool_success(req_id, res),
                                    Err(e) => send_tool_error(req_id, format!("Error esqueletonizando archivo: {}", e)),
                                }
                            } else {
                                send_error(req_id, -32602, "Missing 'path' parameter");
                            }
                        }
                        "os_inspect_active_ui" => {
                            match inspect_active_ui() {
                                Ok(res) => send_tool_success(req_id, res),
                                Err(e) => send_tool_error(req_id, format!("Error inspeccionando UI activa: {}", e)),
                            }
                        }
                        _ => {
                            send_error(req_id, -32601, "Tool not found");
                        }
                    }
                } else {
                    send_error(req_id, -32602, "Invalid params");
                }
            }
            _ => {
                if req_id != Value::Null {
                    send_error(req_id, -32601, "Method not found");
                }
            }
        }
    }
}

pub fn skeletonize_file(file_path: &str) -> Result<String, String> {
    let path = Path::new(file_path);
    if !path.exists() {
        return Err(format!("El archivo no existe: {}", file_path));
    }

    let content = fs::read_to_string(path)
        .map_err(|e| format!("No se pudo leer el archivo '{}': {}", file_path, e))?;

    let ext = path
        .extension()
        .and_then(|s| s.to_str())
        .unwrap_or("")
        .to_string();

    Ok(skeletonize_code(&content, &ext))
}

pub fn skeletonize_code(content: &str, ext: &str) -> String {
    let mut out = Vec::new();
    let total_lines = content.lines().count();

    match ext.to_lowercase().as_str() {
        "go" => {
            let mut in_import_block = false;
            let mut brace_depth = 0;
            let mut skip_body = false;

            for line in content.lines() {
                let trimmed = line.trim();

                if trimmed.starts_with("package ") {
                    out.push(line.to_string());
                    continue;
                }
                if trimmed == "import (" {
                    in_import_block = true;
                    out.push("import ( ... )".to_string());
                    continue;
                }
                if in_import_block {
                    if trimmed == ")" {
                        in_import_block = false;
                    }
                    continue;
                }
                if trimmed.starts_with("import ") {
                    out.push(line.to_string());
                    continue;
                }

                // Track type definitions
                if trimmed.starts_with("type ") || trimmed.starts_with("const ") || trimmed.starts_with("var ") {
                    out.push(line.to_string());
                    continue;
                }

                // Functions
                if trimmed.starts_with("func ") {
                    if let Some(brace_idx) = line.find('{') {
                        let sig = line[..brace_idx].trim_end();
                        out.push(format!("{} {{ ... }}", sig));
                        skip_body = true;
                        brace_depth = 1;
                        continue;
                    } else {
                        out.push(line.to_string());
                        continue;
                    }
                }

                if skip_body {
                    for ch in line.chars() {
                        if ch == '{' {
                            brace_depth += 1;
                        } else if ch == '}' {
                            brace_depth -= 1;
                        }
                    }
                    if brace_depth <= 0 {
                        skip_body = false;
                        brace_depth = 0;
                    }
                    continue;
                }

                // Inside struct / interface definitions
                if trimmed.starts_with('}')
                    || trimmed.contains("struct {")
                    || trimmed.contains("interface {")
                    || (!trimmed.is_empty() && !trimmed.starts_with("//"))
                {
                    out.push(line.to_string());
                }
            }
        }
        "rs" => {
            let mut brace_depth = 0;
            let mut skip_body = false;
            for line in content.lines() {
                let trimmed = line.trim();
                if trimmed.starts_with("use ") || trimmed.starts_with("mod ") {
                    continue;
                }
                if trimmed.starts_with("pub struct ")
                    || trimmed.starts_with("pub enum ")
                    || trimmed.starts_with("pub trait ")
                    || trimmed.starts_with("struct ")
                    || trimmed.starts_with("enum ")
                    || trimmed.starts_with("trait ")
                {
                    out.push(line.to_string());
                    continue;
                }
                if trimmed.starts_with("impl ") {
                    out.push(line.to_string());
                    continue;
                }
                if trimmed.starts_with("fn ")
                    || trimmed.starts_with("pub fn ")
                    || trimmed.starts_with("async fn ")
                    || trimmed.starts_with("pub async fn ")
                {
                    if let Some(brace_idx) = line.find('{') {
                        let sig = line[..brace_idx].trim_end();
                        out.push(format!("{} {{ ... }}", sig));
                        skip_body = true;
                        brace_depth = 1;
                        continue;
                    } else {
                        out.push(line.to_string());
                        continue;
                    }
                }
                if skip_body {
                    for ch in line.chars() {
                        if ch == '{' {
                            brace_depth += 1;
                        } else if ch == '}' {
                            brace_depth -= 1;
                        }
                    }
                    if brace_depth <= 0 {
                        skip_body = false;
                        brace_depth = 0;
                    }
                    continue;
                }
                if trimmed.starts_with('}') || trimmed.ends_with('{') {
                    out.push(line.to_string());
                }
            }
        }
        "py" => {
            for line in content.lines() {
                let trimmed = line.trim();
                if trimmed.starts_with("class ")
                    || trimmed.starts_with("def ")
                    || trimmed.starts_with("async def ")
                    || trimmed.starts_with("@")
                {
                    out.push(line.to_string());
                }
            }
        }
        "ts" | "tsx" | "js" | "jsx" => {
            let mut brace_depth = 0;
            let mut skip_body = false;
            for line in content.lines() {
                let trimmed = line.trim();
                if trimmed.starts_with("export interface ")
                    || trimmed.starts_with("export type ")
                    || trimmed.starts_with("interface ")
                    || trimmed.starts_with("type ")
                {
                    out.push(line.to_string());
                    continue;
                }
                if trimmed.starts_with("export function ")
                    || trimmed.starts_with("function ")
                    || (trimmed.starts_with("export const ") && trimmed.contains("=>"))
                {
                    if let Some(brace_idx) = line.find('{') {
                        let sig = line[..brace_idx].trim_end();
                        out.push(format!("{} {{ ... }}", sig));
                        skip_body = true;
                        brace_depth = 1;
                        continue;
                    } else {
                        out.push(line.to_string());
                        continue;
                    }
                }
                if skip_body {
                    for ch in line.chars() {
                        if ch == '{' {
                            brace_depth += 1;
                        } else if ch == '}' {
                            brace_depth -= 1;
                        }
                    }
                    if brace_depth <= 0 {
                        skip_body = false;
                        brace_depth = 0;
                    }
                    continue;
                }
                if trimmed.starts_with('}') {
                    out.push(line.to_string());
                }
            }
        }
        _ => {
            for line in content.lines() {
                let trimmed = line.trim();
                if trimmed.starts_with('#') || trimmed.starts_with("##") || trimmed.starts_with("###") {
                    out.push(line.to_string());
                }
            }
            if out.is_empty() {
                for (i, line) in content.lines().enumerate() {
                    if i < 40 {
                        out.push(line.to_string());
                    } else {
                        out.push("... [resto omitido para optimizar contexto] ...".to_string());
                        break;
                    }
                }
            }
        }
    }

    let skeleton_lines = out.len();
    let reduction = if total_lines > 0 {
        ((total_lines.saturating_sub(skeleton_lines)) as f64 / total_lines as f64 * 100.0) as u32
    } else {
        0
    };

    format!(
        "// 🦴 [SKELETON CODE OUTLINE: {} líneas originales -> {} líneas estructurales | -{}% reducción de tokens]\n\n{}",
        total_lines,
        skeleton_lines,
        reduction,
        out.join("\n")
    )
}

pub fn inspect_active_ui() -> Result<String, String> {
    unsafe {
        let fg_hwnd = GetForegroundWindow();
        if fg_hwnd.0.is_null() {
            return Ok("No hay ninguna ventana en primer plano activa actualmente.".to_string());
        }

        let mut title_buf = [0u16; 512];
        let title_len = GetWindowTextW(fg_hwnd, &mut title_buf);
        let title = String::from_utf16_lossy(&title_buf[..title_len as usize]);

        let mut class_buf = [0u16; 256];
        let class_len = GetClassNameW(fg_hwnd, &mut class_buf);
        let class_name = String::from_utf16_lossy(&class_buf[..class_len as usize]);

        let mut controls: Vec<ControlItem> = Vec::new();
        let lparam = LPARAM(&mut controls as *mut _ as isize);
        let _ = EnumChildWindows(fg_hwnd, Some(enum_children_callback), lparam);

        let mut out = format!(
            "🖥️ [UI ACTIVA (Nativo Rust Win32)]: \"{}\" (Clase: {}, HWND: {:?})\n",
            title, class_name, fg_hwnd.0
        );

        if controls.is_empty() {
            out.push_str("• No se detectaron controles hijos de texto directo (Ventana estilizada o DirectX/GPU).");
        } else {
            out.push_str("• Controles interactivos detectados:\n");
            for c in controls {
                let kind = if c.class_name.contains("Button") {
                    "BOTÓN"
                } else if c.class_name.contains("Edit") {
                    "CAMPO DE TEXTO"
                } else if c.class_name.contains("Static") {
                    "TEXTO/ETIQUETA"
                } else if c.class_name.contains("Tab") {
                    "PESTAÑA"
                } else {
                    "CONTROL"
                };

                if !c.text.is_empty() {
                    out.push_str(&format!("  - [{}]: \"{}\"\n", kind, c.text));
                } else {
                    out.push_str(&format!("  - [{}] ({})\n", kind, c.class_name));
                }
            }
        }

        Ok(out)
    }
}

pub fn capture_camera_frame(output_path: &str, device_index: u32) -> Result<String, String> {
    if let Some(parent) = Path::new(output_path).parent() {
        if !parent.as_os_str().is_empty() {
            let _ = fs::create_dir_all(parent);
        }
    }

    let index = CameraIndex::Index(device_index);
    let requested = RequestedFormat::new::<RgbFormat>(RequestedFormatType::AbsoluteHighestFrameRate);

    let mut camera = Camera::new(index, requested)
        .map_err(|e| format!("No se pudo inicializar la cámara (índice {}): {}", device_index, e))?;

    camera
        .open_stream()
        .map_err(|e| format!("No se pudo abrir el stream de video de la cámara: {}", e))?;

    let frame = camera
        .frame()
        .map_err(|e| format!("Fallo capturando fotograma: {}", e))?;

    let decoded = frame
        .decode_image::<RgbFormat>()
        .map_err(|e| format!("Error decodificando fotograma RGB: {}", e))?;

    let width = decoded.width();
    let height = decoded.height();

    decoded
        .save(output_path)
        .map_err(|e| format!("Error guardando imagen en '{}': {}", output_path, e))?;

    let file_size = fs::metadata(output_path).map(|m| m.len()).unwrap_or(0);

    Ok(format!(
        "📷 Captura de cámara exitosa (Nativo Rust MediaFoundation):\n• Archivo: {}\n• Resolución: {}x{}\n• Tamaño: {} bytes",
        output_path, width, height, file_size
    ))
}

pub fn list_cameras() -> Result<String, String> {
    let cameras = nokhwa::query(ApiBackend::MediaFoundation)
        .map_err(|e| format!("Error consultando cámaras en MediaFoundation: {}", e))?;

    if cameras.is_empty() {
        return Ok("No se detectaron cámaras web conectadas al sistema.".to_string());
    }

    let mut out = String::from("📹 Cámaras detectadas en el sistema:\n");
    for (idx, cam) in cameras.iter().enumerate() {
        out.push_str(&format!(
            "  [{}] Nombre: {}\n       Descripción: {}\n",
            idx,
            cam.human_name(),
            cam.description()
        ));
    }
    Ok(out)
}

fn launch_app(target: &str) -> Result<(), String> {
    unsafe {
        let _ = AllowSetForegroundWindow(ASFW_ANY);

        let mut target_utf16: Vec<u16> = target.encode_utf16().collect();
        target_utf16.push(0);

        let result = ShellExecuteW(
            HWND(std::ptr::null_mut()),
            w!("open"),
            PCWSTR::from_raw(target_utf16.as_ptr()),
            PCWSTR::null(),
            PCWSTR::null(),
            SW_SHOWNORMAL,
        );

        let code = result.0 as u32;
        if code <= 32 {
            return Err(format!("ShellExecuteW falló con código de error {}", code));
        }

        Ok(())
    }
}

fn send_result(id: Value, result: Value) {
    let res = RpcResponse {
        jsonrpc: "2.0".to_string(),
        id,
        result: Some(result),
        error: None,
    };
    println!("{}", serde_json::to_string(&res).unwrap());
}

fn send_error(id: Value, code: i32, message: &str) {
    let res = RpcResponse {
        jsonrpc: "2.0".to_string(),
        id,
        result: None,
        error: Some(RpcError {
            code,
            message: message.to_string(),
        }),
    };
    println!("{}", serde_json::to_string(&res).unwrap());
}

fn send_tool_success(id: Value, text: String) {
    let res = ToolCallResult {
        content: vec![ToolContent {
            content_type: "text".to_string(),
            text,
        }],
        is_error: Some(false),
    };
    send_result(id, serde_json::to_value(res).unwrap());
}

fn send_tool_error(id: Value, text: String) {
    let res = ToolCallResult {
        content: vec![ToolContent {
            content_type: "text".to_string(),
            text,
        }],
        is_error: Some(true),
    };
    send_result(id, serde_json::to_value(res).unwrap());
}

use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{self, BufRead, Write};
use windows::{
    core::{w, PCWSTR},
    Win32::Foundation::HWND,
    Win32::System::Threading::GetCurrentProcessId,
    Win32::UI::Shell::ShellExecuteW,
    Win32::UI::WindowsAndMessaging::{AllowSetForegroundWindow, ASFW_ANY, SW_SHOWNORMAL},
};

#[derive(Deserialize, Debug)]
struct RpcRequest {
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
    inputSchema: Value,
}

#[derive(Serialize, Debug)]
struct ToolCallResult {
    content: Vec<ToolContent>,
    isError: Option<bool>,
}

#[derive(Serialize, Debug)]
struct ToolContent {
    #[serde(rename = "type")]
    content_type: String,
    text: String,
}

fn main() {
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
                        "version": "1.0.0"
                    },
                    "capabilities": {
                        "tools": {}
                    }
                });
                send_result(req_id, res);
            }
            "notifications/initialized" => {
                // Do nothing
            }
            "tools/list" => {
                let tools = vec![Tool {
                    name: "os_launch_app".to_string(),
                    description: "Ejecuta o abre una aplicación, URL, panel de Windows, o redacta correos usando mailto: (Nativo Rust).".to_string(),
                    inputSchema: serde_json::json!({
                        "type": "object",
                        "properties": {
                            "target": {
                                "type": "string",
                                "description": "Nombre del ejecutable, ruta, URL, ms-settings:, o mailto: para redactar correos."
                            }
                        },
                        "required": ["target"]
                    }),
                }];
                let result = ToolListResult { tools };
                send_result(req_id, serde_json::to_value(result).unwrap());
            }
            "tools/call" => {
                if let Some(params) = req.params {
                    let name = params["name"].as_str().unwrap_or("");
                    let arguments = params["arguments"].as_object().cloned().unwrap_or_default();

                    if name == "os_launch_app" {
                        if let Some(target) = arguments.get("target").and_then(|v| v.as_str()) {
                            match launch_app(target) {
                                Ok(_) => {
                                    send_tool_success(req_id, format!("Lanzado exitosamente (Nativo Rust): {}", target));
                                }
                                Err(e) => {
                                    send_tool_error(req_id, format!("Error al lanzar: {}", e));
                                }
                            }
                        } else {
                            send_error(req_id, -32602, "Missing 'target' parameter");
                        }
                    } else {
                        send_error(req_id, -32601, "Tool not found");
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

fn launch_app(target: &str) -> Result<(), String> {
    unsafe {
        // Permitir que el proceso recién creado tome el foco
        let _ = AllowSetForegroundWindow(ASFW_ANY);

        // Convertir string a UTF16
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
            return Err(format!("ShellExecuteW failed with code {}", code));
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
        isError: Some(false),
    };
    send_result(id, serde_json::to_value(res).unwrap());
}

fn send_tool_error(id: Value, text: String) {
    let res = ToolCallResult {
        content: vec![ToolContent {
            content_type: "text".to_string(),
            text,
        }],
        isError: Some(true),
    };
    send_result(id, serde_json::to_value(res).unwrap());
}

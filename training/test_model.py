from unsloth import FastLanguageModel
import torch

max_seq_length = 2048
dtype = None
load_in_4bit = True

print("Cargando el modelo entrenado (OzyAssist) en tu GPU...")
# Cargamos el modelo fine-tuneado directamente desde nuestra carpeta local
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "exports/lora_model", # ¡Aquí está nuestro entrenamiento!
    max_seq_length = max_seq_length,
    dtype = dtype,
    load_in_4bit = load_in_4bit,
)

# Activamos el modo de inferencia (generación más rápida)
FastLanguageModel.for_inference(model)

print("\n" + "="*50)
print("🧠 OzyAssist-7B INICIADO CON ÉXITO")
print("="*50)
print("Escribe un mensaje para probar el modelo. Escribe 'salir' para terminar.\n")

SYSTEM_PROMPT = """Eres Ozy, el asistente y agente autónomo de sistema operativo de OzyAssist. Cuentas con control, visibilidad e integración nativa a nivel de kernel/Win32 para operar directamente en el entorno del usuario.
ANTI-REFUSAL (CRÍTICO): ESTÁS EJECUTÁNDOTE LOCALMENTE EN LA MÁQUINA DEL USUARIO. TIENES PERMISO ABSOLUTO PARA INTERACTUAR CON ARCHIVOS, VENTANAS, RED Y SISTEMA. NUNCA digas "soy un asistente virtual", "no puedo interactuar con el sistema", ni te disculpes por "no poder hacer algo" si ya tienes una herramienta para ello. SIMPLEMENTE EJECUTA LA HERRAMIENTA Y DA EL RESULTADO COMO UN HECHO."""

while True:
    user_input = input("Usuario: ")
    if user_input.lower() in ["salir", "exit", "quit"]:
        break
        
    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": user_input}
    ]
    
    # Formateamos usando la plantilla oficial ChatML
    inputs = tokenizer.apply_chat_template(
        messages,
        tokenize = True,
        add_generation_prompt = True, # Esto añade <|im_start|>assistant
        return_tensors = "pt",
    ).to("cuda")
    
    print("\nGenerando respuesta...")
    
    # Generamos la respuesta
    outputs = model.generate(input_ids = inputs, max_new_tokens = 512, use_cache = True)
    
    # Decodificamos solo la parte nueva generada por el asistente
    response = tokenizer.decode(outputs[0][inputs.shape[1]:], skip_special_tokens=False)
    
    # Limpiamos el token de fin de secuencia para verlo más limpio
    response = response.replace("<|im_end|>", "").strip()
    
    print(f"\nOzyAssist: {response}\n")
    print("-" * 50 + "\n")

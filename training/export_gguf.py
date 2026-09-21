import os
from unsloth import FastLanguageModel

script_dir = os.path.dirname(os.path.abspath(__file__))
lora_path = os.path.join(script_dir, "exports", "lora_model_v3")
output_gguf = os.path.join(script_dir, "exports", "OzyAssist-7B-v3")

print(f"Cargando el modelo entrenado (Base + LoRA v3) desde: {lora_path}...")
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = lora_path,
    max_seq_length = 2048,
    dtype = None,
    load_in_4bit = True,
)

print(f"Convirtiendo y exportando a formato GGUF q4_k_m en: {output_gguf}...")
model.save_pretrained_gguf(output_gguf, tokenizer, quantization_method = "q4_k_m")

print("¡Exportación completada exitosamente! El modelo ahora puede ser usado.")

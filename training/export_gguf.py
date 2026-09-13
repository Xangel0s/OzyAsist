from unsloth import FastLanguageModel

print("Cargando el modelo entrenado (Base + LoRA)...")
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "exports/lora_model",
    max_seq_length = 2048,
    dtype = None,
    load_in_4bit = True,
)

print("Convirtiendo y exportando a formato GGUF (Ollama compatible)...")
# Exportamos usando el método nativo de Unsloth a q4_k_m (buen balance de peso y rendimiento para 8GB VRAM)
model.save_pretrained_gguf("exports/OzyAssist-7B", tokenizer, quantization_method = "q4_k_m")

print("¡Exportación completada exitosamente! El modelo ahora puede ser usado en Ollama.")

import os
import json
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments
from unsloth import FastLanguageModel, is_bfloat16_supported, get_chat_template

def auto_train_incremental():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    auto_data_path = os.path.join(script_dir, "dataset", "auto_dataset.jsonl")
    base_data_path = os.path.join(script_dir, "dataset", "ozy_distilled_v4.jsonl")
    
    if not os.path.exists(auto_data_path):
        print(f"[INFO] No hay interacciones suficientes aún en: {auto_data_path}")
        return

    with open(auto_data_path, "r", encoding="utf-8") as f:
        new_samples = [line for line in f if line.strip()]

    print(f"=== AUTO-ENTRENAMIENTO INCREMENTAL (Nuevas interacciones de usuario: {len(new_samples)}) ===")
    if len(new_samples) < 5:
        print("[INFO] Se recomiendan al menos 5 interacciones para actualizar el modelo.")
        return

    # Combinar dataset base + interacciones reales de usuario
    combined_path = os.path.join(script_dir, "dataset", "ozy_combined_autotrain.jsonl")
    with open(combined_path, "w", encoding="utf-8") as out_f:
        if os.path.exists(base_data_path):
            with open(base_data_path, "r", encoding="utf-8") as base_f:
                for line in base_f:
                    if line.strip():
                        out_f.write(line)
        # Añadir las del usuario duplicadas 3x para darles peso de adaptación prioritario
        for line in new_samples:
            for _ in range(3):
                out_f.write(line.strip() + "\n")

    output_dir = os.path.join(script_dir, "outputs_autotrain")
    output_gguf = os.path.join(script_dir, "exports", "OzyAssist-3B-v3")

    model, tokenizer = FastLanguageModel.from_pretrained(
        model_name="unsloth/Qwen2.5-Coder-3B-Instruct-bnb-4bit",
        max_seq_length=2048,
        load_in_4bit=True,
    )

    model = FastLanguageModel.get_peft_model(
        model,
        r=16,
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        lora_alpha=16,
        lora_dropout=0,
        bias="none",
        use_gradient_checkpointing="unsloth",
        random_state=3407,
    )

    tokenizer = get_chat_template(tokenizer, chat_template="chatml")

    def formatting_prompts_func(examples):
        convos = examples["messages"]
        texts = [tokenizer.apply_chat_template(c, tokenize=False, add_generation_prompt=False) for c in convos]
        return {"text": texts}

    dataset = load_dataset("json", data_files=combined_path, split="train")
    dataset = dataset.map(formatting_prompts_func, batched=True)

    trainer = SFTTrainer(
        model=model,
        tokenizer=tokenizer,
        train_dataset=dataset,
        dataset_text_field="text",
        max_seq_length=2048,
        packing=False,
        args=TrainingArguments(
            per_device_train_batch_size=1,
            gradient_accumulation_steps=8,
            warmup_steps=2,
            max_steps=20, # Rápido: ~2-3 minutos
            learning_rate=1.5e-4,
            fp16=not is_bfloat16_supported(),
            bf16=is_bfloat16_supported(),
            logging_steps=1,
            optim="adamw_8bit",
            output_dir=output_dir,
        ),
    )

    print("\n--- Ejecutando auto-ajuste LoRA sobre la experiencia del usuario ---")
    trainer.train()

    print(f"\n--- Re-exportando GGUF optimizado a {output_gguf} ---")
    model.save_pretrained_gguf(output_gguf, tokenizer, quantization_method="q4_k_m")
    print("\n[OK] ¡Modelo auto-adaptado con éxito a las interacciones del usuario!")

if __name__ == "__main__":
    auto_train_incremental()

import os
os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["HF_XET_HIGH_PERFORMANCE"] = "0"
from unsloth import FastLanguageModel
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments
from unsloth import is_bfloat16_supported

max_seq_length = 2048
dtype = None
load_in_4bit = True

print("=== INICIANDO ENTRENAMIENTO DE GENERALIZACIÓN: OzyAssist-3B-v5 ===")
print("=== Incluye jergas de LATAM/España, anáforas 'ábrelo' y os_file_info nativo ===")

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Qwen2.5-Coder-3B-Instruct-bnb-4bit",
    max_seq_length = max_seq_length,
    dtype = dtype,
    load_in_4bit = load_in_4bit,
)

# Configurar adaptadores LoRA
model = FastLanguageModel.get_peft_model(
    model,
    r = 16,
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",
                      "gate_proj", "up_proj", "down_proj",],
    lora_alpha = 16,
    lora_dropout = 0,
    bias = "none",
    use_gradient_checkpointing = "unsloth",
    random_state = 3407,
    use_rslora = False,
    loftq_config = None,
)

from unsloth import get_chat_template
tokenizer = get_chat_template(
    tokenizer,
    chat_template = "chatml",
)

def formatting_prompts_func(examples):
    convos = examples["messages"]
    texts = [tokenizer.apply_chat_template(convo, tokenize = False, add_generation_prompt = False) for convo in convos]
    return { "text" : texts }

script_dir = os.path.dirname(os.path.abspath(__file__))
dataset_path = os.path.join(script_dir, "dataset", "ozy_distilled_v5.jsonl")
output_lora = os.path.join(script_dir, "exports", "lora_model_3b_v5")
output_dir = os.path.join(script_dir, "outputs_3b_v5")
output_gguf = os.path.join(script_dir, "exports", "OzyAssist-3B-v5")

dataset = load_dataset("json", data_files=dataset_path, split="train", download_mode="force_redownload")
dataset = dataset.map(formatting_prompts_func, batched = True)

trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    dataset_text_field = "text",
    max_seq_length = max_seq_length,
    packing = False,
    args = TrainingArguments(
        per_device_train_batch_size = 1,
        gradient_accumulation_steps = 8,
        warmup_steps = 10,
        max_steps = 120,
        learning_rate = 2.0e-4,
        fp16 = not is_bfloat16_supported(),
        bf16 = is_bfloat16_supported(),
        logging_steps = 5,
        optim = "adamw_8bit",
        weight_decay = 0.01,
        lr_scheduler_type = "cosine",
        seed = 3407,
        output_dir = output_dir,
    ),
)

print(f"\n--- Entrenando OzyAssist-3B-v5 sobre {len(dataset)} ejemplos diversos en GPU ---")
trainer_stats = trainer.train()

print(f"\n--- Guardando adaptadores LoRA en {output_lora} ---")
model.save_pretrained(output_lora)
tokenizer.save_pretrained(output_lora)

print(f"\n--- Exportando a GGUF Q4_K_M en {output_gguf} ---")
model.save_pretrained_gguf(output_gguf, tokenizer, quantization_method = "q4_k_m")

print("\n[OK] ¡OzyAssist-3B-v5 entrenado y exportado exitosamente a GGUF Q4_K_M!")

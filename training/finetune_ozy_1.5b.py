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

print("=== INICIANDO ENTRENAMIENTO DE DESTILACIÓN: OzyAssist-1.5B (Ultra-Fast Voice & Stream) ===")

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Qwen2.5-Coder-1.5B-Instruct-bnb-4bit",
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
dataset_path = os.path.join(script_dir, "dataset", "ozy_dataset_v3.jsonl")
output_lora = os.path.join(script_dir, "exports", "lora_model_1.5b")
output_dir = os.path.join(script_dir, "outputs_1.5b")
output_gguf = os.path.join(script_dir, "exports", "OzyAssist-1.5B-v3")

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
        per_device_train_batch_size = 2,
        gradient_accumulation_steps = 4,
        warmup_steps = 5,
        max_steps = 60, # ~2.3 épocas en 1.5B (super rápido)
        learning_rate = 3e-4,
        fp16 = not is_bfloat16_supported(),
        bf16 = is_bfloat16_supported(),
        logging_steps = 1,
        optim = "adamw_8bit",
        weight_decay = 0.01,
        lr_scheduler_type = "linear",
        seed = 3407,
        output_dir = output_dir,
    ),
)

print("\n--- Entrenando OzyAssist-1.5B en GPU ---")
trainer_stats = trainer.train()

print(f"\n--- Guardando adaptadores LoRA en {output_lora} ---")
model.save_pretrained(output_lora)
tokenizer.save_pretrained(output_lora)

print(f"\n--- Exportando a GGUF Q4_K_M en {output_gguf} ---")
model.save_pretrained_gguf(output_gguf, tokenizer, quantization_method = "q4_k_m")

print("\n[OK] ¡OzyAssist-1.5B entrenado y exportado exitosamente!")

import os
os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["HF_XET_HIGH_PERFORMANCE"] = "0"
from unsloth import FastLanguageModel
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments
from unsloth import is_bfloat16_supported

max_seq_length = 2048 # Soportamos 2048 por defecto para 8GB VRAM
dtype = None
load_in_4bit = True # Para VRAM de 8GB esto es obligatorio

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Qwen2.5-Coder-7B-Instruct-bnb-4bit",
    max_seq_length = max_seq_length,
    dtype = dtype,
    load_in_4bit = load_in_4bit,
)

# Aplicar adaptadores LoRA (Esto es lo que entrenaremos)
model = FastLanguageModel.get_peft_model(
    model,
    r = 16, # Rank, 16 o 32 son buenos valores
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",
                      "gate_proj", "up_proj", "down_proj",],
    lora_alpha = 16,
    lora_dropout = 0, # Unsloth soporta 0 drop-out
    bias = "none",
    use_gradient_checkpointing = "unsloth",
    random_state = 3407,
    use_rslora = False,
    loftq_config = None,
)

from unsloth import get_chat_template
tokenizer = get_chat_template(
    tokenizer,
    chat_template = "chatml", # Qwen2.5 usa ChatML
)

def formatting_prompts_func(examples):
    convos = examples["messages"]
    texts = [tokenizer.apply_chat_template(convo, tokenize = False, add_generation_prompt = False) for convo in convos]
    return { "text" : texts }

dataset = load_dataset("json", data_files="dataset/ozy_dataset_v2.jsonl", split="train", download_mode="force_redownload")
dataset = dataset.map(formatting_prompts_func, batched = True)

trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    dataset_text_field = "text",
    max_seq_length = max_seq_length,
    packing = False, # Puede acelerar el entrenamiento si es True, pero consume más VRAM
    args = TrainingArguments(
        per_device_train_batch_size = 1,
        gradient_accumulation_steps = 8,
        warmup_steps = 5,
        max_steps = 60, # Corto para pruebas, subir a 500+ para el real
        learning_rate = 2e-4,
        fp16 = not is_bfloat16_supported(),
        bf16 = is_bfloat16_supported(),
        logging_steps = 1,
        optim = "adamw_8bit",
        weight_decay = 0.01,
        lr_scheduler_type = "linear",
        seed = 3407,
        output_dir = "outputs",
    ),
)

# Empezar el entrenamiento
print("Iniciando Fine-Tuning de OzyAssist...")
trainer_stats = trainer.train()

# Guardar el modelo localmente en formato LoRA
model.save_pretrained("exports/lora_model")
tokenizer.save_pretrained("exports/lora_model")

print("Fine-tuning completado y guardado en exports/lora_model")

# NOTA: Para convertir a GGUF usar unsloth (opcional, requiere llama.cpp)
# model.save_pretrained_gguf("exports/OzyAssist-7B-Agent", tokenizer, quantization_method = "q4_k_m")

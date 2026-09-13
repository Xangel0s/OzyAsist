import sys
import logging
logging.basicConfig(level=logging.DEBUG)

print("Iniciando importación...")
from unsloth import FastLanguageModel
print("Importación terminada. Iniciando descarga...")

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Qwen2.5-Coder-7B-Instruct-bnb-4bit", # Versión oficial de unsloth en 4bit
    max_seq_length = 2048,
    dtype = None,
    load_in_4bit = True,
)
print("¡Carga exitosa!")

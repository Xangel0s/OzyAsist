#!/usr/bin/env python3
"""
Offline STT Helper using cached whisper-tiny in training venv
"""
import sys
import os
import json
import warnings
warnings.filterwarnings("ignore")

def main():
    if len(sys.argv) < 2:
        print("")
        sys.exit(1)
    
    audio_path = sys.argv[1]
    if not os.path.exists(audio_path):
        print("")
        sys.exit(1)

    try:
        import torch
        from transformers import pipeline
        device = "cuda:0" if torch.cuda.is_available() else "cpu"
        pipe = pipeline(
            "automatic-speech-recognition",
            model="openai/whisper-tiny",
            device=device,
            generate_kwargs={"language": "es", "task": "transcribe"}
        )
        res = pipe(audio_path)
        text = res.get("text", "").strip()
        print(text)
    except Exception as e:
        sys.stderr.write(f"STT Error: {e}\n")
        sys.exit(1)

if __name__ == "__main__":
    main()

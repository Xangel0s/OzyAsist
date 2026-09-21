from huggingface_hub import HfApi
api = HfApi()
try:
    user = api.whoami()
    print("HF User:", user["name"])
    models = [m.id for m in api.list_models(author=user["name"])]
    print("Models:", models)
    for m in models:
        print(f"\nFiles in {m}:")
        print(api.list_repo_files(m))
except Exception as e:
    print("Error:", e)

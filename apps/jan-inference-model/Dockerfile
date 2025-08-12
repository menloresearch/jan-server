FROM vllm/vllm-openai:latest

RUN pip install --no-cache-dir huggingface_hub \
 && python3 -c "from huggingface_hub import snapshot_download; \
                snapshot_download(repo_id='Qwen/Qwen3-4B', local_dir='/models/Qwen3-4B')"

EXPOSE 8101

CMD ["--model", "Qwen/Qwen3-4B", "--host", "0.0.0.0", "--port", "8101", "--gpu-memory-utilization", "0.65", "--enforce-eager", "--max_model_len", "32768"]
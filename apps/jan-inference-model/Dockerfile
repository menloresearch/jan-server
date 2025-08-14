FROM vllm/vllm-openai:v0.10.0

RUN hf download janhq/Jan-v1-4B --local-dir /models/Jan-v1-4B

EXPOSE 8101

CMD ["--model", "/models/Jan-v1-4B", \
    "--served-model-name", "jan-v1-4b", \
    "--host", "0.0.0.0", \
    "--port", "8101", \
    "--max-num-batched-tokens", "1024", \
    "--enable-auto-tool-choice", \
    "--tool-call-parser", "hermes", \
    "--reasoning-parser", "qwen3"]

┌─────────────────────────────────────────────────────────────┐
│                    INTERFACE LAYER                          │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  completions.go                                        │ │
│  │  - HTTP request/response handling                      │ │
│  │  - Uses ChatUseCase + StreamingService                 │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    DOMAIN LAYER                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  ChatUseCase                                          │ │
│  │  - Business logic orchestration                         │ │
│  │  - Request validation                                  │ │
│  │  - Non-streaming completions                           │ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  StreamingService                                     │ │
│  │  - Two-channel streaming logic                         │ │
│  │  - HTTP context handling                               │ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  InferenceProvider Interface                          │ │
│  │  - Abstraction for external services                   │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                 INFRASTRUCTURE LAYER                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  JanInferenceProvider                                 │ │
│  │  - Implements InferenceProvider interface              │ │
│  │  - Handles Jan Inference API calls                    │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘


application/app/
├── domain/
│   ├── chat/
│   │   ├── chat_usecase.go      # Business logic orchestration
│   │   ├── streaming_service.go  # Streaming-specific logic
│   │   └── constants.go          # Shared constants
│   └── inference/
│       └── inference_provider.go # Domain interface
├── infrastructure/
│   ├── inference/
│   │   └── jan_inference_provider.go # Jan Inference implementation
│   └── infrastructure_provider.go    # Infrastructure providers
└── interfaces/http/routes/v1/chat/
    └── completions.go            # HTTP handlers (updated)
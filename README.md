# Jan Server
Self-hosted jan.ai 

# Requiremnts
- docker
- minikube
- helm

# Setup
- minikube start
- eval $(minikube docker-env)
- alias kubectl="minikube kubectl --"
- ./scripts/run.sh
# Services Architecture
[![](https://mermaid.ink/img/pako:eNp9j09PwkAQxb_KZM5Q2P6DNtFEaWIwISGRk5TDph1otd3F7RbElu_utoCJB93Lzm8y772ZBhOZEoa4LeQxybjSsIpiAebNipyEhuHwvq1IpKDoo6ZKVy08c_GwnD9xTUd-agyBQbjy-aL-NdOZQDs6sJGJ0KNElvuCdC5FBcdcZ1CaHQq4gzcuhn3d9mUutqRIJHRprruk-a0Hi663-TetTPYVtGC-YUXqkCdUrRezJbxc4S91O7IsS-qMlLk2Wq8eow0OcKfyFMMtLyoaYEmq5B1j05nEaKZLijE0ZcrVe4yxOBvRnotXKUsMtaqNTMl6l92g3qcmM8r5TvHyx9lcl5KayVpoDB3P6z0wbPDToG9bPrMZs21_wsbBdIAnDG3bs5jree7YYSwIXH9yHuBXnzq2nMCfTqaOw6ZB4Lt2cP4G0qegKQ?type=png)](https://mermaid.live/edit#pako:eNp9j09PwkAQxb_KZM5Q2P6DNtFEaWIwISGRk5TDph1otd3F7RbElu_utoCJB93Lzm8y772ZBhOZEoa4LeQxybjSsIpiAebNipyEhuHwvq1IpKDoo6ZKVy08c_GwnD9xTUd-agyBQbjy-aL-NdOZQDs6sJGJ0KNElvuCdC5FBcdcZ1CaHQq4gzcuhn3d9mUutqRIJHRprruk-a0Hi663-TetTPYVtGC-YUXqkCdUrRezJbxc4S91O7IsS-qMlLk2Wq8eow0OcKfyFMMtLyoaYEmq5B1j05nEaKZLijE0ZcrVe4yxOBvRnotXKUsMtaqNTMl6l92g3qcmM8r5TvHyx9lcl5KayVpoDB3P6z0wbPDToG9bPrMZs21_wsbBdIAnDG3bs5jree7YYSwIXH9yHuBXnzq2nMCfTqaOw6ZB4Lt2cP4G0qegKQ)

[Mermaid](https://mermaid.live/edit#pako:eNp9j09PwkAQxb_KZM5Q2P6DNtFEaWIwISGRk5TDph1otd3F7RbElu_utoCJB93Lzm8y772ZBhOZEoa4LeQxybjSsIpiAebNipyEhuHwvq1IpKDoo6ZKVy08c_GwnD9xTUd-agyBQbjy-aL-NdOZQDs6sJGJ0KNElvuCdC5FBcdcZ1CaHQq4gzcuhn3d9mUutqRIJHRprruk-a0Hi663-TetTPYVtGC-YUXqkCdUrRezJbxc4S91O7IsS-qMlLk2Wq8eow0OcKfyFMMtLyoaYEmq5B1j05nEaKZLijE0ZcrVe4yxOBvRnotXKUsMtaqNTMl6l92g3qcmM8r5TvHyx9lcl5KayVpoDB3P6z0wbPDToG9bPrMZs21_wsbBdIAnDG3bs5jree7YYSwIXH9yHuBXnzq2nMCfTqaOw6ZB4Lt2cP4G0qegKQ)

# [Configurable Environment variables](https://github.com/menloresearch/jan-server/blob/main/charts/umbrella-chart/values.yaml)
- jan-api-gateway:
    - SERPER_API_KEY: 
        - API keys from SERPER_API
    - OAUTH2_GOOGLE_CLIENT_ID
    - OAUTH2_GOOGLE_CLIENT_SECRET
    - OAUTH2_GOOGLE_REDIRECT_URL
        - Secrets from Google OAuth2
    - JWT_SECRET:
        - HMAC-SHA-256
    - APIKEY_SECRET
        - HMAC-SHA256
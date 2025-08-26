# Release Procedure

## 0. Announce Release
- Create a thread in **#jan-server-internal** channel  
  - `Jan Server Release vX.X.X (Release Manager:)`  

## 2. Pre-flight Checks
- **System Environment** 
  - Confirm environment variables are correctly configured  
  - jan-api-gateway:
    - git diff release..main -- apps/jan-api-gateway/application/config/environment_variables/env.go
  - Ask DevOps to verify **Prod Vault** is updated 

- **Database**  
  - Review **migration plan**  
  - Confirm migrations are backward compatible (or plan downtime)  

- **API Compatibility**  
  - Validate no **breaking changes** for clients (or prepare migration guides)  

## 3. Deployment
- Merge `main` → `release` branch  
- Tag release with target version: vX.X.X
- Use the GitHub UI for tagging, it generates some useful notes.

**Monitoring**
- /v1/versions
  ```bash
  curl -X 'GET' \
  'https://api.jan.ai/v1/version' \
  -H 'accept: application/json'
  ```
- Run Prod Test cases

# Object Storage Configuration

To enable persistent Object Storage instead of local files:

## Set Environment Variables in Replit Secrets:

1. `GOOGLE_CLOUD_PROJECT_ID` - Your Google Cloud Project ID
2. `OBJECT_STORAGE_BUCKET` - Your bucket name (e.g., "bus-app-data")
3. `GOOGLE_APPLICATION_CREDENTIALS_JSON` - Your service account JSON credentials

## Local Development Fallback:

If these environment variables are not set, the app will automatically fall back to using local `data/` folder files. This ensures the app works in both development and production environments.

## Migration Process:

1. Set up your Object Storage credentials in Replit Secrets
2. The app will automatically start using Object Storage for new data
3. Your existing JSON files in `data/` folder serve as backup/reference

## Benefits:

- ✅ Data persists across deployments and restarts
- ✅ No data loss when redeploying
- ✅ Automatic fallback to local storage for development
- ✅ Same JSON structure and API

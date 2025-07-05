#!/bin/bash
# Deploy to Cloud Run with environment variables

echo "🚀 Deploying TruckPe to Cloud Run..."

if gcloud run deploy truckpe-backend \
  --source . \
  --region=us-central1 \
  --env-vars-file .env.yaml \
  --update-env-vars USE_MEMORY_STORE=true \
  --allow-unauthenticated; then
    echo "✅ Deployment complete!"
    echo "📱 Test with WhatsApp: +1 415 523 8886"
    echo "🔗 Service URL: https://truckpe-backend-153285185067.us-central1.run.app"
else
    echo "❌ Deployment failed! Check logs with:"
    echo "gcloud run services logs read truckpe-backend --region=us-central1 --limit=20"
    exit 1
fi
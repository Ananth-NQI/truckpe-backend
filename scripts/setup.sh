#!/bin/bash

echo "🚀 TruckPe First Time Setup"
echo "=========================="

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

# Check gcloud installation
if ! command -v gcloud &> /dev/null; then
    echo "❌ gcloud CLI is not installed. Please install Google Cloud SDK."
    exit 1
fi

# Create .gitignore if not exists
if [ ! -f .gitignore ]; then
    cat > .gitignore << 'GITIGNORE'
# Environment files
.env
.env.*
!.env.example
!environments/.env.example

# Binaries
cloud_sql_proxy
scripts/cloud_sql_proxy
main
*.exe

# IDE
.vscode/
.idea/

# OS
.DS_Store

# Test coverage
*.out
coverage.html
GITIGNORE
fi

# Install dependencies
echo "📦 Installing Go dependencies..."
go mod download

# Create example env file
cat > environments/.env.example << 'EXAMPLE'
# Example Environment Configuration
APP_NAME=TruckPe
ENVIRONMENT=development

# Storage
USE_MEMORY_STORE=true

# Server
PORT=8080

# Twilio Configuration
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_WHATSAPP_FROM=whatsapp:+14155238886

# Feature Flags
ENABLE_TEMPLATE_TESTING=true
ENABLE_DEBUG_LOGS=true
EXAMPLE

echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Run 'make dev' to start development server"
echo "2. Run 'make help' to see all available commands"

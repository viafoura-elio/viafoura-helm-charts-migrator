#!/bin/bash

# SOPS Helper Script for Managing Secrets
# Usage: ./sops-helper.sh <encrypt|decrypt|edit> <account> [cluster] [namespace]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Check if SOPS is installed
if ! command -v sops &> /dev/null; then
    print_color $RED "Error: SOPS is not installed"
    echo "Install SOPS: brew install sops"
    exit 1
fi

# Parse arguments
if [ $# -lt 2 ]; then
    print_color $RED "Error: Insufficient arguments"
    echo ""
    echo "Usage: $0 <encrypt|decrypt|edit> <account> [cluster] [namespace]"
    echo ""
    echo "Examples:"
    echo "  $0 encrypt testdev                              # Encrypt account level secrets"
    echo "  $0 encrypt testdev dev01                        # Encrypt cluster level secrets"
    echo "  $0 encrypt testdev dev01 vf-dev2                # Encrypt namespace level secrets"
    echo "  $0 decrypt production                           # Decrypt account level secrets"
    echo "  $0 edit testdev test01 vf-test3                 # Edit encrypted secrets directly"
    echo ""
    echo "Available accounts:"
    ls -d */ 2>/dev/null | grep -v README | xargs -n1 basename || echo "  No accounts found"
    exit 1
fi

ACTION=$1
ACCOUNT=$2
CLUSTER=$3
NAMESPACE=$4

# Determine the path based on provided arguments
if [ -n "$NAMESPACE" ] && [ -n "$CLUSTER" ]; then
    # Namespace level
    BASE_PATH="${ACCOUNT}/clusters/${CLUSTER}/namespaces/${NAMESPACE}"
    LEVEL="Namespace"
    LEVEL_NAME="${NAMESPACE}"
elif [ -n "$CLUSTER" ]; then
    # Cluster level
    BASE_PATH="${ACCOUNT}/clusters/${CLUSTER}"
    LEVEL="Cluster"
    LEVEL_NAME="${CLUSTER}"
else
    # Account level
    BASE_PATH="${ACCOUNT}"
    LEVEL="Account"
    LEVEL_NAME="${ACCOUNT}"
fi

DEC_FILE="${BASE_PATH}/secrets.dec.yaml"
ENC_FILE="${BASE_PATH}/secrets.enc.yaml"

print_color $BLUE "=================================================================================="
print_color $BLUE "SOPS Secrets Management"
print_color $BLUE "=================================================================================="
echo ""
print_color $YELLOW "Action:     ${ACTION}"
print_color $YELLOW "Level:      ${LEVEL}"
print_color $YELLOW "Name:       ${LEVEL_NAME}"
print_color $YELLOW "Path:       ${BASE_PATH}"
echo ""

# Function to encrypt secrets
encrypt_secrets() {
    if [ ! -f "$DEC_FILE" ]; then
        print_color $RED "Error: Decrypted file not found: $DEC_FILE"
        exit 1
    fi

    print_color $GREEN "Encrypting ${LEVEL} level secrets..."

    # Create backup of existing encrypted file if it exists
    if [ -f "$ENC_FILE" ]; then
        cp "$ENC_FILE" "${ENC_FILE}.backup"
        print_color $YELLOW "Created backup: ${ENC_FILE}.backup"
    fi

    # Encrypt the file
    if sops -e "$DEC_FILE" > "$ENC_FILE"; then
        print_color $GREEN "✓ Successfully encrypted: $ENC_FILE"

        # Verify encryption
        if sops -d "$ENC_FILE" > /dev/null 2>&1; then
            print_color $GREEN "✓ Encryption verified"
        else
            print_color $RED "⚠ Warning: Unable to verify encryption"
        fi
    else
        print_color $RED "✗ Encryption failed"
        # Restore backup if encryption failed
        if [ -f "${ENC_FILE}.backup" ]; then
            mv "${ENC_FILE}.backup" "$ENC_FILE"
            print_color $YELLOW "Restored backup"
        fi
        exit 1
    fi
}

# Function to decrypt secrets
decrypt_secrets() {
    if [ ! -f "$ENC_FILE" ]; then
        print_color $RED "Error: Encrypted file not found: $ENC_FILE"
        exit 1
    fi

    print_color $GREEN "Decrypting ${LEVEL} level secrets..."

    # Create backup of existing decrypted file if it exists
    if [ -f "$DEC_FILE" ]; then
        cp "$DEC_FILE" "${DEC_FILE}.backup"
        print_color $YELLOW "Created backup: ${DEC_FILE}.backup"
    fi

    # Decrypt the file
    if sops -d "$ENC_FILE" > "$DEC_FILE.tmp"; then
        mv "$DEC_FILE.tmp" "$DEC_FILE"
        print_color $GREEN "✓ Successfully decrypted: $DEC_FILE"

        # Show warning about plain text secrets
        print_color $YELLOW "⚠ WARNING: Decrypted secrets are now in plain text!"
        print_color $YELLOW "  Remember to:"
        print_color $YELLOW "  1. Re-encrypt after making changes"
        print_color $YELLOW "  2. Never commit decrypted files to git"
        print_color $YELLOW "  3. Add *.dec.yaml to .gitignore"
    else
        print_color $RED "✗ Decryption failed"
        # Restore backup if decryption failed
        if [ -f "${DEC_FILE}.backup" ]; then
            mv "${DEC_FILE}.backup" "$DEC_FILE"
            print_color $YELLOW "Restored backup"
        fi
        exit 1
    fi
}

# Function to edit encrypted secrets directly
edit_secrets() {
    if [ ! -f "$ENC_FILE" ]; then
        print_color $RED "Error: Encrypted file not found: $ENC_FILE"
        exit 1
    fi

    print_color $GREEN "Editing ${LEVEL} level secrets..."
    print_color $YELLOW "Opening encrypted file in editor..."

    # Edit the encrypted file directly with SOPS
    if sops "$ENC_FILE"; then
        print_color $GREEN "✓ Successfully saved changes"
    else
        print_color $RED "✗ Edit cancelled or failed"
        exit 1
    fi
}

# Execute the requested action
case $ACTION in
    encrypt|enc|e)
        encrypt_secrets
        ;;
    decrypt|dec|d)
        decrypt_secrets
        ;;
    edit|ed)
        edit_secrets
        ;;
    *)
        print_color $RED "Error: Invalid action: $ACTION"
        echo "Valid actions: encrypt, decrypt, edit"
        exit 1
        ;;
esac

echo ""
print_color $BLUE "=================================================================================="
print_color $GREEN "Operation completed successfully!"
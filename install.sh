#!/bin/bash

echo ""
echo " ____  __     __  ____   ___   __   ____  ____"
echo "(  _ \(  )   /  \(  _ \ / __) / _\ / ___)(_  _)"
echo " ) _ (/ (_/\(  O )) _ (( (__ /    \\___ \  )("
echo "(____/\____/ \__/(____/ \___)\_/\_/(____/ (__)"
echo ""

# Declare a log file and a temp directory
LOGFILE="$HOME/blobcast-temp/logfile.log"
TEMP_DIR="$HOME/blobcast-temp"

# Parse command line arguments
while getopts "v:" opt; do
  case $opt in
    v) USER_VERSION="$OPTARG"
    ;;
    \?) echo "Invalid option -$OPTARG" >&2
        echo "Usage: $0 [-v version]"
        echo "Example: $0 -v v0.4.0"
        exit 1
    ;;
  esac
done

# Check if the directory exists
if [ -d "$TEMP_DIR" ]; then
    read -p "Directory $TEMP_DIR exists. Do you want to clear it out? (y/n) " -n 1 -r
    echo    # move to a new line
    if [[ $REPLY =~ ^[Yy]$ ]]
    then
        rm -rf "$TEMP_DIR"
        echo "Directory $TEMP_DIR has been removed."
    fi
fi

# Create a temporary directory to work from
mkdir -p "$TEMP_DIR"
touch "$LOGFILE"

# Log and print the log file location
echo "Log file is located at: $LOGFILE" | tee -a "$LOGFILE"

# Change to $TEMP_DIR and print a message
cd "$TEMP_DIR" || exit 1
echo "Working from temporary directory: $TEMP_DIR" | tee -a "$LOGFILE"

# Set VERSION based on user input or fetch latest
if [ -n "$USER_VERSION" ]; then
    VERSION="$USER_VERSION"
    echo "Using specified version: $VERSION" | tee -a "$LOGFILE"
else
    # Fetch the latest release tag from GitHub
    VERSION=$(curl -s "https://api.github.com/repos/forma-dev/blobcast/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    # Check if VERSION is empty
    if [ -z "$VERSION" ]; then
        echo "Failed to fetch the latest version. Exiting." | tee -a "$LOGFILE"
        exit 1
    fi
    echo "Using latest version: $VERSION" | tee -a "$LOGFILE"
fi

# Validate version format
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo "Invalid version format: $VERSION" | tee -a "$LOGFILE"
    echo "Version should start with vX.X.X (e.g., v0.4.0)" | tee -a "$LOGFILE"
    exit 1
fi

# Detect the operating system and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Translate architecture to expected format
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH. Exiting." | tee -a "$LOGFILE"
        exit 1
        ;;
esac

# Translate OS to expected format
case $OS in
    Linux)
        OS="linux"
        ;;
    Darwin)
        OS="darwin"
        ;;
    *)
        echo "Unsupported operating system: $OS. Exiting." | tee -a "$LOGFILE"
        exit 1
        ;;
esac

# Construct the download URL
PLATFORM="${OS}-${ARCH}"
URL="https://github.com/forma-dev/blobcast/releases/download/$VERSION/blobcast-${PLATFORM}"

# Check if URL is valid
if [[ ! $URL =~ ^https://github.com/forma-dev/blobcast/releases/download/[^/]+/blobcast-[^/]+$ ]]; then
    echo "Invalid URL: $URL. Exiting." | tee -a "$LOGFILE"
    exit 1
fi

# Log and print a message
echo "Downloading from: $URL" | tee -a "$LOGFILE"

# Download the tarball
if ! curl -L "$URL" -o "blobcast-${PLATFORM}" >> "$LOGFILE" 2>&1; then
    echo "Download failed. Exiting." | tee -a "$LOGFILE"
    exit 1
fi

# Detect if running on macOS and use appropriate command for checksum
if [ "$OS" = "Darwin" ]; then
    CALCULATED_CHECKSUM=$(shasum -a 256 "blobcast-${PLATFORM}" | awk '{print $1}')
else
    CALCULATED_CHECKSUM=$(sha256sum "blobcast-${PLATFORM}" | awk '{print $1}')
fi

# Download checksums
if ! curl -L "https://github.com/forma-dev/blobcast/releases/download/$VERSION/checksums-${OS}-${ARCH}.txt" -o "checksums.txt" >> "$LOGFILE" 2>&1; then
    echo "Failed to download checksums. Exiting." | tee -a "$LOGFILE"
    exit 1
fi

# Find the expected checksum in checksums.txt
EXPECTED_CHECKSUM=$(grep "blobcast-${PLATFORM}" checksums.txt | awk '{print $1}')

# Verify the checksum
if [ "$CALCULATED_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "Checksum verification failed. Expected: $EXPECTED_CHECKSUM, but got: $CALCULATED_CHECKSUM. Exiting." | tee -a "$LOGFILE"
    exit 1
else
    echo "Checksum verification successful." | tee -a "$LOGFILE"
fi

# Rename the binary to blobcast
mv "blobcast-${PLATFORM}" "blobcast"

# Remove the checksums to clean up
rm "checksums.txt"

# Log and print a message
echo "Temporary files cleaned up." | tee -a "$LOGFILE"

# Ask the user where to install the binary
echo ""
echo "Where would you like to install the blobcast binary?"
echo "1) System bin directory (/usr/local/bin) [Recommended]"
echo "2) Keep in current directory ($TEMP_DIR)"

read -p "Enter your choice: " -n 1 -r
echo

case $REPLY in
    1)
        # Install to /usr/local/bin
        chmod +x "$TEMP_DIR/blobcast"
        sudo mv "$TEMP_DIR/blobcast" /usr/local/bin/
        echo "Binary moved to /usr/local/bin" | tee -a "$LOGFILE"
        echo ""
        echo "You can now run blobcast from anywhere." | tee -a "$LOGFILE"
        ;;
    2)
        # Keep in current directory
        chmod +x "$TEMP_DIR/blobcast"
        echo "Binary kept in $TEMP_DIR" | tee -a "$LOGFILE"
        echo "You can run blobcast from this directory using ./blobcast" | tee -a "$LOGFILE"
        ;;
    *)
        echo ""
        echo "Invalid choice. The binary remains in $TEMP_DIR" | tee -a "$LOGFILE"
        chmod +x "$TEMP_DIR/blobcast"
        echo "You can run blobcast from this directory using ./blobcast" | tee -a "$LOGFILE"
        ;;
esac

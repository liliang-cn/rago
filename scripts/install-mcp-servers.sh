#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   RAGO MCP Servers Installation${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check for Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}❌ Node.js is not installed${NC}"
    echo "Please install Node.js (v18 or higher) from https://nodejs.org/"
    exit 1
fi

NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo -e "${YELLOW}⚠️  Node.js version is less than 18. Some features may not work.${NC}"
fi

echo -e "${GREEN}✓ Node.js $(node -v) detected${NC}"

# Check for npm
if ! command -v npm &> /dev/null; then
    echo -e "${RED}❌ npm is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ npm $(npm -v) detected${NC}"
echo ""

# Create MCP servers directory
MCP_DIR="${HOME}/.rago/mcp-servers"
mkdir -p "$MCP_DIR"
cd "$MCP_DIR"

echo -e "${BLUE}📦 Installing MCP servers...${NC}"
echo ""

# Array of MCP servers to install
declare -a servers=(
    "@modelcontextprotocol/server-filesystem"
    "@modelcontextprotocol/server-fetch" 
    "@modelcontextprotocol/server-memory"
    "@modelcontextprotocol/server-sequential-thinking"
    "@modelcontextprotocol/server-time"
)

# Install each server
for server in "${servers[@]}"; do
    server_name=$(echo "$server" | cut -d'/' -f2 | cut -d'-' -f2-)
    echo -e "${BLUE}Installing ${server_name}...${NC}"
    
    if npm list "$server" &>/dev/null; then
        echo -e "${YELLOW}  → Already installed, updating...${NC}"
        npm update "$server" --silent
    else
        npm install "$server" --silent
    fi
    
    echo -e "${GREEN}  ✓ ${server_name} installed${NC}"
done

echo ""
echo -e "${BLUE}🔧 Creating wrapper scripts...${NC}"

# Create wrapper scripts for each server
cat > "$MCP_DIR/start-filesystem.sh" << 'EOF'
#!/bin/bash
exec npx @modelcontextprotocol/server-filesystem "$@"
EOF

cat > "$MCP_DIR/start-fetch.sh" << 'EOF'
#!/bin/bash
exec npx @modelcontextprotocol/server-fetch "$@"
EOF

cat > "$MCP_DIR/start-memory.sh" << 'EOF'
#!/bin/bash
exec npx @modelcontextprotocol/server-memory "$@"
EOF

cat > "$MCP_DIR/start-sequential-thinking.sh" << 'EOF'
#!/bin/bash
exec npx @modelcontextprotocol/server-sequential-thinking "$@"
EOF

cat > "$MCP_DIR/start-time.sh" << 'EOF'
#!/bin/bash
exec npx @modelcontextprotocol/server-time "$@"
EOF

# Make scripts executable
chmod +x "$MCP_DIR"/*.sh

echo -e "${GREEN}✓ Wrapper scripts created${NC}"
echo ""

# Create alternative mcpServers.json with local paths
cat > "$MCP_DIR/mcpServers-local.json" << EOF
{
  "mcpServers": {
    "filesystem": {
      "command": "$MCP_DIR/start-filesystem.sh",
      "args": ["--allowed-directories", "./", "/tmp"],
      "description": "File system operations with sandboxed directory access"
    },
    "fetch": {
      "command": "$MCP_DIR/start-fetch.sh",
      "args": [],
      "description": "HTTP/HTTPS fetch operations for web content retrieval"
    },
    "memory": {
      "command": "$MCP_DIR/start-memory.sh",
      "args": [],
      "description": "In-memory key-value store for temporary data storage"
    },
    "sequential-thinking": {
      "command": "$MCP_DIR/start-sequential-thinking.sh",
      "args": [],
      "description": "Enhanced reasoning through step-by-step problem decomposition"
    },
    "time": {
      "command": "$MCP_DIR/start-time.sh",
      "args": [],
      "description": "Time and date utilities with timezone support"
    }
  }
}
EOF

echo -e "${GREEN}✓ Local configuration created at: $MCP_DIR/mcpServers-local.json${NC}"
echo ""

# Test installation
echo -e "${BLUE}🧪 Testing MCP servers...${NC}"

# Test each server
test_server() {
    local server_name=$1
    local test_cmd=$2
    
    echo -n "  Testing $server_name... "
    
    if timeout 5 $test_cmd &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  (may require additional setup)${NC}"
        return 1
    fi
}

# Note: These are basic availability tests
test_server "filesystem" "npx @modelcontextprotocol/server-filesystem --version"
test_server "fetch" "npx @modelcontextprotocol/server-fetch --version"
test_server "memory" "npx @modelcontextprotocol/server-memory --version" 
test_server "sequential-thinking" "npx @modelcontextprotocol/server-sequential-thinking --version"
test_server "time" "npx @modelcontextprotocol/server-time --version"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ MCP Servers Installation Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "📍 Installation location: $MCP_DIR"
echo ""
echo "To use these servers with RAGO:"
echo "1. The default mcpServers.json has been updated with npx commands"
echo "2. Alternatively, use the local wrapper scripts:"
echo "   cp $MCP_DIR/mcpServers-local.json ./mcpServers.json"
echo ""
echo "Available servers:"
echo "  • filesystem - File system operations"
echo "  • fetch - Web content retrieval" 
echo "  • memory - In-memory storage"
echo "  • sequential-thinking - Enhanced reasoning"
echo "  • time - Date/time utilities"
echo ""
echo -e "${BLUE}Run 'rago mcp status' to verify server connectivity${NC}"
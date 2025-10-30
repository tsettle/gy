#!/bin/bash
# Demo script showing gy capabilities with test files

set -e

echo "=== gy Test & Demo Suite ==="
echo ""

# Use GY environment variable or default to ./gy
GY=${GY:-./gy}

# Check if gy exists
if ! command -v "$GY" &> /dev/null; then
    echo "Error: gy not found at '$GY'. Build it first with: go build"
    echo "Or set GY environment variable to point to gy binary"
    exit 1
fi

cd "$(dirname "$0")"

echo "--- 1. Basic Path Extraction (simple.yml) ---"
echo "$ $GY 'database.host' test/simple.yml"
"$GY" 'database.host' test/simple.yml
echo ""

echo "--- 2. Trim Mode - Just the Value (simple.yml) ---"
echo "$ $GY -t 'database.port' test/simple.yml"
"$GY" -t 'database.port' test/simple.yml
echo ""

echo "--- 3. Nested Object Extraction (simple.yml) ---"
echo "$ $GY 'database.credentials' test/simple.yml"
"$GY" 'database.credentials' test/simple.yml
echo ""

echo "--- 4. Array Indexing (arrays.yml) ---"
echo "$ $GY 'users[0].name' test/arrays.yml"
"$GY" 'users[0].name' test/arrays.yml
echo ""

echo "--- 5. Array with Trim (arrays.yml) ---"
echo "$ $GY -t 'users[1].email' test/arrays.yml"
"$GY" -t 'users[1].email' test/arrays.yml
echo ""

echo "--- 6. Nested Array Access (arrays.yml) ---"
echo "$ $GY -t 'users[0].roles[0]' test/arrays.yml"
"$GY" -t 'users[0].roles[0]' test/arrays.yml
echo ""

echo "--- 7. List Mode - Explore Structure (simple.yml) ---"
echo "$ $GY -l 'database' test/simple.yml"
"$GY" -l 'database' test/simple.yml
echo ""

echo "--- 8. List with Depth (snmp.yml) ---"
echo "$ $GY -l --depth 2 'modules' test/snmp.yml"
"$GY" -l --depth 2 'modules' test/snmp.yml
echo ""

echo "--- 9. Complex Path (snmp.yml) ---"
echo "$ $GY -t 'modules.if_mib.walk[0]' test/snmp.yml"
"$GY" -t 'modules.if_mib.walk[0]' test/snmp.yml
echo ""

echo "--- 10. Kubernetes-style Manifest (kubernetes.yml) ---"
echo "$ $GY 'spec.template.spec.containers[0].name' test/kubernetes.yml"
"$GY" 'spec.template.spec.containers[0].name' test/kubernetes.yml
echo ""

echo "--- 11. Environment Variables (kubernetes.yml) ---"
echo "$ $GY -t 'spec.template.spec.containers[0].env[0].value' test/kubernetes.yml"
"$GY" -t 'spec.template.spec.containers[0].env[0].value' test/kubernetes.yml
echo ""

echo "--- 12. Type Preservation (types.yml) ---"
echo "$ $GY 'numbers' test/types.yml"
"$GY" 'numbers' test/types.yml
echo ""

echo "--- 13. Special Keys - Numeric (types.yml) ---"
echo "$ $GY -t 'special_keys.1' test/types.yml"
"$GY" -t 'special_keys.1' test/types.yml
echo ""

echo "--- 14. Boolean Values (types.yml) ---"
echo "$ $GY 'booleans' test/types.yml"
"$GY" 'booleans' test/types.yml
echo ""

echo "--- 15. Piping from stdin ---"
echo "$ cat test/simple.yml | $GY 'app.name'"
cat test/simple.yml | "$GY" 'app.name'
echo ""

echo "--- 16. Ansible Playbook - Variables (ansible.yml) ---"
echo "$ $GY -t '[0].vars.app_name' test/ansible.yml"
"$GY" -t '[0].vars.app_name' test/ansible.yml
echo ""

echo "--- 17. Ansible - Role Configuration (ansible.yml) ---"
echo "$ $GY '[0].roles[1].vars' test/ansible.yml"
"$GY" '[0].roles[1].vars' test/ansible.yml
echo ""

echo "--- 18. Ansible - Task by Index (ansible.yml) ---"
echo "$ $GY -t '[0].tasks[0].name' test/ansible.yml"
"$GY" -t '[0].tasks[0].name' test/ansible.yml
echo ""

echo "--- 19. Ansible - Handler Names (ansible.yml) ---"
echo "$ $GY -l '[0].handlers' test/ansible.yml"
"$GY" -l '[0].handlers' test/ansible.yml
echo ""

echo "--- 20. Ansible - Nested Loop Items (ansible.yml) ---"
echo "$ $GY -t '[0].tasks[7].loop[1]' test/ansible.yml"
"$GY" -t '[0].tasks[7].loop[1]' test/ansible.yml
echo ""

echo "=== All tests completed ==="
#!/bin/bash
# Demo script showing gy capabilities with test files

set -e

echo "=== gy Test & Demo Suite ==="
echo ""

# Check if gy exists
if ! command -v gy &> /dev/null; then
    echo "Error: gy not found. Build it first with: go build gy.go"
    exit 1
fi

cd "$(dirname "$0")"

echo "--- 1. Basic Path Extraction (simple.yml) ---"
echo "$ gy 'database.host' test/simple.yml"
gy 'database.host' test/simple.yml
echo ""

echo "--- 2. Trim Mode - Just the Value (simple.yml) ---"
echo "$ gy -t 'database.port' test/simple.yml"
gy -t 'database.port' test/simple.yml
echo ""

echo "--- 3. Nested Object Extraction (simple.yml) ---"
echo "$ gy 'database.credentials' test/simple.yml"
gy 'database.credentials' test/simple.yml
echo ""

echo "--- 4. Array Indexing (arrays.yml) ---"
echo "$ gy 'users[0].name' test/arrays.yml"
gy 'users[0].name' test/arrays.yml
echo ""

echo "--- 5. Array with Trim (arrays.yml) ---"
echo "$ gy -t 'users[1].email' test/arrays.yml"
gy -t 'users[1].email' test/arrays.yml
echo ""

echo "--- 6. Nested Array Access (arrays.yml) ---"
echo "$ gy -t 'users[0].roles[0]' test/arrays.yml"
gy -t 'users[0].roles[0]' test/arrays.yml
echo ""

echo "--- 7. List Mode - Explore Structure (simple.yml) ---"
echo "$ gy -l 'database' test/simple.yml"
gy -l 'database' test/simple.yml
echo ""

echo "--- 8. List with Depth (snmp.yml) ---"
echo "$ gy -l --depth 2 'modules' test/snmp.yml"
gy -l --depth 2 'modules' test/snmp.yml
echo ""

echo "--- 9. Complex Path (snmp.yml) ---"
echo "$ gy -t 'modules.if_mib.walk[0]' test/snmp.yml"
gy -t 'modules.if_mib.walk[0]' test/snmp.yml
echo ""

echo "--- 10. Kubernetes-style Manifest (kubernetes.yml) ---"
echo "$ gy 'spec.template.spec.containers[0].name' test/kubernetes.yml"
gy 'spec.template.spec.containers[0].name' test/kubernetes.yml
echo ""

echo "--- 11. Environment Variables (kubernetes.yml) ---"
echo "$ gy -t 'spec.template.spec.containers[0].env[0].value' test/kubernetes.yml"
gy -t 'spec.template.spec.containers[0].env[0].value' test/kubernetes.yml
echo ""

echo "--- 12. Type Preservation (types.yml) ---"
echo "$ gy 'numbers' test/types.yml"
gy 'numbers' test/types.yml
echo ""

echo "--- 13. Special Keys - Numeric (types.yml) ---"
echo "$ gy -t 'special_keys.1' test/types.yml"
gy -t 'special_keys.1' test/types.yml
echo ""

echo "--- 14. Boolean Values (types.yml) ---"
echo "$ gy 'booleans' test/types.yml"
gy 'booleans' test/types.yml
echo ""

echo "--- 15. Piping from stdin ---"
echo "$ cat test/simple.yml | gy 'app.name'"
cat test/simple.yml | gy 'app.name'
echo ""

echo "--- 16. Ansible Playbook - Variables (ansible.yml) ---"
echo "$ gy -t '[0].vars.app_name' test/ansible.yml"
gy -t '[0].vars.app_name' test/ansible.yml
echo ""

echo "--- 17. Ansible - Role Configuration (ansible.yml) ---"
echo "$ gy '[0].roles[1].vars' test/ansible.yml"
gy '[0].roles[1].vars' test/ansible.yml
echo ""

echo "--- 18. Ansible - Task by Index (ansible.yml) ---"
echo "$ gy -t '[0].tasks[0].name' test/ansible.yml"
gy -t '[0].tasks[0].name' test/ansible.yml
echo ""

echo "--- 19. Ansible - Handler Names (ansible.yml) ---"
echo "$ gy -l '[0].handlers' test/ansible.yml"
gy -l '[0].handlers' test/ansible.yml
echo ""

echo "--- 20. Ansible - Nested Loop Items (ansible.yml) ---"
echo "$ gy -t '[0].tasks[8].loop[1]' test/ansible.yml"
gy -t '[0].tasks[8].loop[1]' test/ansible.yml
echo ""

echo "=== All tests completed successfully! ==="

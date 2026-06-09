import os
import re
import sys

# Regex to detect common API keys / secrets / credentials
SECRET_PATTERNS = [
    r'(?i)(api_key|secret_key|private_key|password|jwt_secret|token)\s*[:=]\s*["\'][a-zA-Z0-9_\-\.\~]{8,}["\']',
]

def scan_file_for_secrets(file_path):
    issues = []
    try:
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            for line_no, line in enumerate(f, 1):
                # Ignore comment lines or template strings that aren't real secrets
                if line.strip().startswith(('#', '//', '/*')):
                    continue
                for pattern in SECRET_PATTERNS:
                    if re.search(pattern, line):
                        # Mask the matched value
                        issues.append(f"Line {line_no}: Potential hardcoded secret found: {line.strip()[:30]}...")
    except Exception as e:
        pass
    return issues

def scan_file_for_todos(file_path):
    todos = []
    try:
        with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
            for line_no, line in enumerate(f, 1):
                if 'TODO:' in line or 'FIXME:' in line:
                    todos.append(f"Line {line_no}: {line.strip()}")
    except Exception as e:
        pass
    return todos

def main():
    workspace_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '../..'))
    print(f"🔍 Running Quality & Security Checklist on: {workspace_dir}")
    print("=" * 60)

    secret_issues = {}
    todo_items = {}
    scanned_files_count = 0

    # Exclude directories
    exclude_dirs = {'.git', 'node_modules', '.next', 'dist', 'build', '.agents'}

    for root, dirs, files in os.walk(workspace_dir):
        # Modify dirs in-place to skip excluded directories
        dirs[:] = [d for d in dirs if d not in exclude_dirs]
        for file in files:
            # Only check source and configuration files
            if not file.endswith(('.ts', '.tsx', '.js', '.jsx', '.py', '.go', '.json', '.env')):
                continue
            
            file_path = os.path.join(root, file)
            rel_path = os.path.relpath(file_path, workspace_dir)
            scanned_files_count += 1

            secrets = scan_file_for_secrets(file_path)
            if secrets:
                secret_issues[rel_path] = secrets

            todos = scan_file_for_todos(file_path)
            if todos:
                todo_items[rel_path] = todos

    print(f"📊 Scanned {scanned_files_count} files.")
    print("-" * 60)

    has_failures = False

    if secret_issues:
        print("❌ SECURITY ALERTS (Potential Hardcoded Secrets):")
        for file, alerts in secret_issues.items():
            print(f"  📂 {file}:")
            for alert in alerts:
                print(f"    - {alert}")
        print("-" * 60)
        has_failures = True
    else:
        print("✅ Security Check: No hardcoded secrets detected in source files.")

    if todo_items:
        print("ℹ️ PENDING TASKS (TODOs / FIXMEs):")
        for file, tasks in todo_items.items():
            print(f"  📂 {file}:")
            for task in tasks:
                print(f"    - {task}")
        print("-" * 60)

    if has_failures:
        print("⚠️ Checklist completed with failures. Please fix the security alerts before deploying.")
        sys.exit(1)
    else:
        print("🎉 All checklist validation checks passed!")
        sys.exit(0)

if __name__ == '__main__':
    main()

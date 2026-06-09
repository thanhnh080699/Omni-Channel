import os
import subprocess
import sys
import json

def run_command(command, cwd):
    print(f"🚀 Running: {' '.join(command)}")
    try:
        result = subprocess.run(command, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=True)
        return True, result.stdout
    except subprocess.CalledProcessError as e:
        return False, f"Stdout:\n{e.stdout}\nStderr:\n{e.stderr}"
    except Exception as e:
        return False, str(e)

def main():
    workspace_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '../..'))
    print("=" * 60)
    print(f"🛡️  Starting Comprehensive Verification on: {workspace_dir}")
    print("=" * 60)

    # 1. Run local security & checklist script first
    checklist_script = os.path.join(os.path.dirname(__file__), 'checklist.py')
    checklist_ok, checklist_output = run_command([sys.executable, checklist_script], workspace_dir)
    print(checklist_output)
    
    if not checklist_ok:
        print("❌ Verification halted: Security / checklist checks failed.")
        sys.exit(1)

    print("-" * 60)
    
    # 2. Tech stack specific checks
    package_json_path = os.path.join(workspace_dir, 'package.json')
    if os.path.exists(package_json_path):
        print("📦 Node.js / JavaScript project detected.")
        try:
            with open(package_json_path, 'r', encoding='utf-8') as f:
                package_data = json.load(f)
            scripts = package_data.get('scripts', {})
            
            # Check for lint script
            if 'lint' in scripts:
                print("📝 Running project linter...")
                lint_ok, lint_out = run_command(['npm', 'run', 'lint'], workspace_dir)
                if not lint_ok:
                    print("❌ Linter checks failed:")
                    print(lint_out)
                    sys.exit(1)
                print("✅ Linter passed.")
            
            # Check for test script
            if 'test' in scripts:
                print("🧪 Running unit tests...")
                test_ok, test_out = run_command(['npm', 'run', 'test'], workspace_dir)
                if not test_ok:
                    print("❌ Unit tests failed:")
                    print(test_out)
                    sys.exit(1)
                print("✅ All unit tests passed.")
                
        except Exception as e:
            print(f"❌ Failed to parse or verify package.json: {e}")
            sys.exit(1)

    # Python specific checks
    elif os.path.exists(os.path.join(workspace_dir, 'pytest.ini')) or os.path.exists(os.path.join(workspace_dir, 'tests')):
        print("🐍 Python project detected.")
        # Check if pytest is available
        print("🧪 Running pytest unit tests...")
        test_ok, test_out = run_command(['pytest'], workspace_dir)
        if not test_ok:
            print("❌ Python pytest unit tests failed:")
            print(test_out)
            sys.exit(1)
        print("✅ All Python unit tests passed.")

    else:
        print("ℹ️ No standard JS/TS or Python project build/test scripts detected. Skipping framework-specific tests.")

    print("=" * 60)
    print("🎉 Verification Completed: All systems go! Ready for deployment.")
    print("=" * 60)
    sys.exit(0)

if __name__ == '__main__':
    main()

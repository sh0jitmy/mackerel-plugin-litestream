import os
import sys
import argparse

LICENSE_HEADER = """// Copyright 2026 [Copyright Holder]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: [YOUR_NAME]
"""

def get_go_files():
    go_files = []
    # 除外ディレクトリ
    exclude_dirs = {'.git', '.github', '.claude', 'dist', 'bin', 'vendor', 'ent', 'ogen'}
    
    for root, dirs, files in os.walk('.'):
        # 除外設定
        dirs[:] = [d for d in dirs if d not in exclude_dirs]
        for file in files:
            if file.endswith('.go'):
                go_files.append(os.path.join(root, file))
    return go_files

def has_license(content):
    # シンプルに主要なキーワードが含まれているかでチェック
    return "Licensed under the Apache License, Version 2.0" in content and "Author:" in content

def check_licenses():
    go_files = get_go_files()
    missing_files = []
    
    for file_path in go_files:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        if not has_license(content):
            missing_files.append(file_path)
            
    if missing_files:
        print("❌ The following Go files are missing the required license & author header:")
        for file in missing_files:
            print(f"  - {file}")
        print("\n👉 Run 'make license-add' to automatically add the headers.")
        return False
    
    print("✅ All Go files have the valid license & author header.")
    return True

def add_licenses():
    go_files = get_go_files()
    added_count = 0
    
    for file_path in go_files:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            
        if not has_license(content):
            # ヘッダーを先頭に挿入
            # もしファイルがすでに // で始まっているなど、ゴミが混ざるのを防ぐため、
            # 単純に先頭に挿入します。
            new_content = LICENSE_HEADER + "\n" + content
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(new_content)
            print(f"✅ Added license header to {file_path}")
            added_count += 1
            
    print(f"🎉 Completed! Added headers to {added_count} files.")
    return True

def main():
    parser = argparse.ArgumentParser(description="Check or Add License & Author headers to Go source files.")
    parser.add_argument('--check', action='store_true', help="Check if files have the header.")
    parser.add_argument('--add', action='store_true', help="Automatically add headers to files missing them.")
    
    args = parser.parse_args()
    
    if args.add:
        add_licenses()
        sys.exit(0)
    elif args.check:
        if check_licenses():
            sys.exit(0)
        else:
            sys.exit(1)
    else:
        # デフォルトはチェック動作
        if check_licenses():
            sys.exit(0)
        else:
            sys.exit(1)

if __name__ == '__main__':
    main()

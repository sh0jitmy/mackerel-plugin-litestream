import os
import re

def self_eval():
    req_file = 'REQUIREMENTS.md'
    if not os.path.exists(req_file):
        print(f"Error: {req_file} not found.")
        return False
    
    with open(req_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Count checked vs unchecked boxes in the requirements section
    # Matches '- [ ]' and '- [x]' or '- [X]'
    lines = content.split('\n')
    
    checked_count = 0
    unchecked_count = 0
    
    # We only count checkboxes under "## 📋 要件チェックリスト" and before "## 📈 自己評価結果"
    in_checklist = False
    for line in lines:
        if '## 📋 要件チェックリスト' in line:
            in_checklist = True
            continue
        if '## 📈 自己評価結果' in line:
            in_checklist = False
            continue
        
        if in_checklist:
            if re.match(r'^\s*-\s*\[x\]', line, re.IGNORECASE):
                checked_count += 1
            elif re.match(r'^\s*-\s*\[\s*\]', line):
                unchecked_count += 1
                
    total = checked_count + unchecked_count
    percentage = (checked_count / total) * 100 if total > 0 else 0
    
    print(f"\n=================== Self-Evaluation Summary ===================")
    print(f"Total Requirements: {total}")
    print(f"Achieved Requirements: {checked_count}")
    print(f"Compliance Rate: {percentage:.2f}%")
    print(f"===============================================================\n")
    
    # Update the summary section in REQUIREMENTS.md
    new_content = content
    # Replace achievement count
    new_content = re.sub(
        r'- \*\*達成要件数\*\*:\s*.*',
        f'- **達成要件数**: {checked_count} / {total}',
        new_content
    )
    # Replace suitability percentage
    new_content = re.sub(
        r'- \*\*適合率 \(達成数/\d+\)\*\*:\s*.*',
        f'- **適合率 (達成数/{total})**: {percentage:.2f} %',
        new_content
    )
    
    with open(req_file, 'w', encoding='utf-8') as f:
        f.write(new_content)
        
    print(f"Successfully updated {req_file} with current scores!")
    return True

if __name__ == '__main__':
    self_eval()

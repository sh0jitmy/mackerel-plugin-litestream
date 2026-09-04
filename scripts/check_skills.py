import os
import re

def check_skills():
    skills_dir = '.claude/skills'
    if not os.path.exists(skills_dir):
        print(f"Error: Directory {skills_dir} not found.")
        return False
    
    success = True
    for item in os.listdir(skills_dir):
        item_path = os.path.join(skills_dir, item)
        if os.path.isdir(item_path):
            skill_file = os.path.join(item_path, 'SKILL.md')
            if not os.path.exists(skill_file):
                print(f"❌ Missing SKILL.md in {item_path}")
                success = False
                continue
            
            try:
                with open(skill_file, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                # Extract frontmatter
                fm_match = re.match(r'^---\s*\n(.*?)\n---\s*\n', content, re.DOTALL)
                if not fm_match:
                    print(f"❌ Frontmatter delimiter '---' not found in {skill_file}")
                    success = False
                    continue
                
                fm_text = fm_match.group(1)
                
                # Parse simple YAML line-by-line using regex
                fm_data = {}
                in_metadata = False
                metadata_data = {}
                
                for line in fm_text.split('\n'):
                    line = line.strip()
                    if not line or line.startswith('#'):
                        continue
                    
                    if line.startswith('metadata:'):
                        in_metadata = True
                        continue
                    
                    if in_metadata and line.startswith('openclaw:'):
                        # Exit metadata block if entering openclaw
                        in_metadata = False
                    
                    # If line has less indentation, we might have exited metadata
                    # but simple splitting works for our static frontmatter
                    if ':' in line:
                        k, v = line.split(':', 1)
                        k = k.strip()
                        v = v.strip().strip('"').strip("'")
                        
                        if in_metadata:
                            metadata_data[k] = v
                        else:
                            fm_data[k] = v
                
                # Check fields
                name = fm_data.get('name')
                if not name:
                    print(f"❌ name field missing in {skill_file}")
                    success = False
                elif name != item:
                    print(f"❌ name field '{name}' does not match directory '{item}' in {skill_file}")
                    success = False
                
                desc = fm_data.get('description')
                if not desc:
                    print(f"❌ description field missing in {skill_file}")
                    success = False
                
                user_invocable = fm_data.get('user-invocable')
                if user_invocable is None:
                    print(f"❌ user-invocable field missing in {skill_file}")
                    success = False
                
                version = metadata_data.get('version')
                if not version:
                    print(f"❌ metadata.version missing in {skill_file}")
                    success = False
                
                allowed_tools = fm_data.get('allowed-tools')
                if not allowed_tools:
                    print(f"❌ allowed-tools field missing in {skill_file}")
                    success = False
                
                if success:
                    print(f"✅ {item}/SKILL.md format is valid.")
                
            except Exception as e:
                print(f"❌ Error parsing {skill_file}: {e}")
                success = False

    return success

if __name__ == '__main__':
    import sys
    if check_skills():
        print("🎉 All custom skills passed checks successfully!")
        sys.exit(0)
    else:
        print("💥 Skill format validation failed.")
        sys.exit(1)

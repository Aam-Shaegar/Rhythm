#!/usr/bin/env python3
"""Create local configuration with random secrets. Never overwrite existing files."""
from pathlib import Path
import os
import secrets

root = Path(__file__).resolve().parent.parent
for target in (root / '.env', root / 'frontend/.env'):
    content = target.with_name(target.name + '.example').read_text()
    if target == root / '.env':
        for key in ('JWT_ACCESS_SECRET', 'JWT_REFRESH_SECRET', 'POSTGRES_PASSWORD'):
            content = '\n'.join(
                key + '=' + secrets.token_hex(32) if line.startswith(key + '=') else line
                for line in content.splitlines()
            ) + '\n'
    try:
        fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError:
        print(f'Keeping existing {target.relative_to(root)}')
        continue
    with os.fdopen(fd, 'w') as out:
        out.write(content)
    print(f'Created {target.relative_to(root)}')

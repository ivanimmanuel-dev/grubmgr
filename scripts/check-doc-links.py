#!/usr/bin/env python3
"""Check local Markdown links in a source or installed documentation directory."""
import pathlib
import re
import sys
from urllib.parse import unquote


def check(files):
    paths = set(files)
    errors = []
    for name, data in files.items():
        if not name.endswith('.md'):
            continue
        text = data.decode('utf-8')
        for link in re.findall(r'\]\(([^()\s]+)\)', text):
            if re.match(r'[a-zA-Z][a-zA-Z0-9+.-]*:', link):
                continue
            target, _, anchor = unquote(link).partition('#')
            parts = list(pathlib.PurePosixPath(name).parent.parts)
            for part in pathlib.PurePosixPath(target).parts:
                if part == '..':
                    if parts:
                        parts.pop()
                    else:
                        errors.append(f'{name}: link escapes documentation root: {link}')
                elif part != '.':
                    parts.append(part)
            destination = '/'.join(parts) if target else name
            if destination not in paths and not any(p.startswith(destination.rstrip('/')+'/') for p in paths):
                errors.append(f'{name}: missing link target: {link}')
            elif anchor and destination.endswith('.md') and destination in files:
                headings = re.findall(r'^#{1,6} (.+)$', files[destination].decode('utf-8'), re.M)
                anchors = {re.sub(r'[^\w\- ]', '', h.lower()).replace(' ', '-') for h in headings}
                if anchor not in anchors:
                    errors.append(f'{name}: missing heading: {link}')
    if errors:
        raise ValueError('\n'.join(errors))


if __name__ == '__main__':
    root = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else '.')
    names = [root/'README.md', root/'THIRD_PARTY_NOTICES.md']
    for directory in ['docs', 'examples', 'third_party']:
        names.extend(p for p in (root/directory).rglob('*') if p.is_file())
    names.append(root/('LICENSE' if (root/'LICENSE').exists() else 'copyright'))
    check({p.relative_to(root).as_posix(): p.read_bytes() for p in names if p.is_file()})
    print('Documentation links verified')

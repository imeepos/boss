#!/usr/bin/env python3
"""合并多个单页 .drawio 为一个多页 .drawio。
用法: merge_pages.py <out.drawio> <in1.drawio> <in2.drawio> ...
页面名取自各输入文件的 diagram name。"""
import sys, re, xml.etree.ElementTree as ET

def main():
    out, ins = sys.argv[1], sys.argv[2:]
    pages = []
    for f in ins:
        t = ET.parse(f)
        d = t.getroot().find('diagram')
        name = d.get('name') or f.split('/')[-1].replace('.drawio', '')
        pages.append((name, ET.tostring(d, encoding='unicode')))
    body = '\n'.join(p for _, p in pages)
    xml = ('<?xml version="1.0" encoding="UTF-8"?>\n<mxfile host="drawio" version="31.1.8">\n'
           + body + '\n</mxfile>\n')
    open(out, 'w').write(xml)
    print('wrote', out, len(pages), 'pages')

if __name__ == '__main__':
    main()

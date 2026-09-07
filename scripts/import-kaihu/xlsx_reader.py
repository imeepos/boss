# -*- coding: utf-8 -*-
# xlsx 只读解析(zipfile + xml.etree 纯 stdlib):返回表名/表头/有效数据行,纯格式空行跳过并计数。
import re
import zipfile
import xml.etree.ElementTree as ET

NS_M = 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'
TAG = '{%s}' % NS_M


def _shared_strings(zf):
    try:
        raw = zf.read('xl/sharedStrings.xml')
    except KeyError:
        return []
    root = ET.fromstring(raw)
    return [''.join(t.text or '' for t in si.iter(TAG + 't')) for si in root.iter(TAG + 'si')]


def _cell_text(cell, strings):
    v = cell.find(TAG + 'v')
    if v is None or v.text is None:
        return ''
    if cell.get('t') == 's':
        return strings[int(v.text)]
    return v.text


def _col_of(ref):
    m = re.match('[A-Z]+', ref)
    return m.group(0) if m else ''


def read_rows(path):
    # 返回 (sheet_names, header, rows, skipped_empty);rows 元素含 _row 原始行号。
    with zipfile.ZipFile(path) as zf:
        wb = ET.fromstring(zf.read('xl/workbook.xml'))
        sheet_names = [el.get('name') for el in wb.iter(TAG + 'sheet')]
        sheet = ET.fromstring(zf.read('xl/worksheets/sheet1.xml'))
        strings = _shared_strings(zf)
    header = {}
    rows = []
    skipped_empty = 0
    for row in sheet.iter(TAG + 'row'):
        rn = int(row.get('r'))
        if rn == 1:
            for c in row.iter(TAG + 'c'):
                col = _col_of(c.get('r'))
                if col:
                    header[col] = _cell_text(c, strings).strip()
            continue
        cells = {}
        for c in row.iter(TAG + 'c'):
            col = _col_of(c.get('r'))
            if col:
                cells[col] = _cell_text(c, strings).strip()
        if any(cells.values()):
            cells['_row'] = rn
            rows.append(cells)
        else:
            skipped_empty += 1
    return sheet_names, header, rows, skipped_empty
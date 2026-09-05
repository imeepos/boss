#!/usr/bin/env python3
"""GB11643 18 位证件号校验位补全: 输入 17 位主体码, 输出含校验位的证件号。
用途: role-demand-sim 注册类场景造数(校验位错误会被实名校验 42200 拒绝)。"""
import sys

b = sys.argv[1]
w = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2]
m = "10X98765432"
s = sum(int(b[i]) * w[i] for i in range(17))
print(b + m[s % 11])

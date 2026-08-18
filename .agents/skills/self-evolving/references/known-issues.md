# Known Issues

<!-- 格式：症状 → 原因 → 修法。排查超过 5 分钟的 bug 才值得记。 -->

## edit 报 "edit requires reading the file first"，但该文件明明看过

症状 → 用 bash `cat` 看过文件内容后调用 edit/write 覆盖，被拒："edit requires reading ... first — read the file, then retry"。
原因 → edit/write 只认 Read 工具的观察记录，bash 输出不算（同一会话内两次踩中）。
修法 → 要编辑/覆盖的文件一律先用 Read 工具读一遍。

## edit 报 "old_string was not found"，内容肉眼完全一致

症状 → old_string 与文件末尾段落逐字符相同却匹配失败。
原因 → old_string 末尾带了 `\n`，而目标文件没有结尾换行。
修法 → 匹配文件末尾段落时去掉 old_string 的尾换行。

## 同一 CDP profile 连拍两套主题截图，"亮色"图拍成了暗色

症状 → 第二次运行截图脚本时，本应亮色的截图呈暗色。
原因 → `--user-data-dir` 复用，localStorage 里上一轮的 `boss.theme=dark` 仍在，首屏内联脚本按它渲染。
修法 → 每个状态显式 `localStorage.setItem` + reload 后再拍；或每次运行 `mktemp -d` 新 profile。

## 脚本收尾 rmSync 临时 profile 报 ENOTEMPTY

症状 → `rmSync(profile, {recursive:true, force:true})` 抛 `ENOTEMPTY, Directory not empty`。
原因 → `proc.kill('SIGTERM')` 后 Chrome 仍在写 profile 目录，立刻 rm 产生竞态。
修法 → kill 后先 `await Promise.race([once(proc,'exit'), sleep(2000)])` 再 rm（scripts/cdp-capture.mjs 已修）。

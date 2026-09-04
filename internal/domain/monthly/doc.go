// Package monthly 月度填报事实域(BI 经营分析,迁移 000181)。
// 权威输入:docs/books/模板_月度填报.xlsx(_RegionList 51 Barangay+三张录入表,
// 灰色列=公式派生勿填)与同目录 3 个标准 CSV(UTF-8 with BOM,CRLF 行尾,表头带单位后缀)。
// 三事实表粒度均为 月×区域;派生列(期末在用/主营总收入)服务端计算,
// 导入与接口一律忽略外部传入的派生值。
package monthly

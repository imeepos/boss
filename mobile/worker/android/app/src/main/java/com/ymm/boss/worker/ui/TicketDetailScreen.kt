package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.R
import com.ymm.boss.worker.api.Api
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

// 非组合上下文取资源(协程 toast 文案)
private fun s(res: Int, vararg fmt: Any = emptyArray()) = Api.context().getString(res, *fmt)

/**
 * 工单详情(对齐 designs/worker-order-detail-v1.spec.md):
 * - 4 屏状态分支由 status 字段决定:TODO/DOING/DONE
 * - 类型(INSTALL/REPAIR)由 stages.length 推断(后端 TicketDetail 暂未返 type)
 * - 卡片实现见 TicketDetailCards.kt
 *
 * 下拉刷新:PageRefresh 容器包裹 LazyColumn,触发器连入 loadOnce 的 key;
 * 回退/重试/领取工单操作成功后调用 nav.requestRefresh() 自动重拉数据,
 * 不必师傅手动下拉。后端若真的回退了 stage,UI 即时反映最新进度。
 */
@Composable
fun TicketDetailScreen(nav: NavHost, no: String) {
    val state by loadOnce(no, nav.refreshTick) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize()) {
        when (val s = state) {
            is Load.Loading -> Box(Modifier.weight(1f)) { Loading() }
            is Load.Fail -> {
                Box(Modifier.weight(1f)) {
                    Column(Modifier.fillMaxSize()) {
                        TopBar(stringResource(R.string.td_title), onBack = { nav.pop() }, action = stringResource(R.string.td_contact_dispatch),
                            onAction = { nav.push(Screen.Service) })
                        Card(Modifier.padding(14.dp)) {
                            Notice(stringResource(R.string.err_ticket_load), red = true)
                        }
                    }
                }
            }
            is Load.Ok -> {
                val d = s.data
                val status = d.optString("status")
                val type = inferTicketType(d)
                val title = if (type == "REPAIR") stringResource(R.string.td_title_repair) else stringResource(R.string.td_title)

                PageRefresh(nav, modifier = Modifier.weight(1f)) {
                    LazyColumn(modifier = Modifier.fillMaxSize()) {
                        item {
                            TopBar(title, onBack = { nav.pop() }, action = stringResource(R.string.td_contact_dispatch),
                                onAction = { nav.push(Screen.Service) })
                            DetailHeaderCard(d, type)
                        }
                        if (status != "TODO") item { QuickActionRow(d, nav, no) }
                        item { TimelineCard(d) }
                        if (status == "DOING" && type == "INSTALL") {
                            item {
                                TimelineExtras(no,
                                    onRollback = {
                                        scope.launch {
                                            try {
                                                TicketApi.rollback(no)
                                                toast(ctx, s(R.string.toast_rollback_sent))
                                                nav.requestRefresh()
                                            } catch (_: Exception) { toast(ctx, s(R.string.err_action)) }
                                        }
                                    },
                                    onRetry = {
                                        scope.launch {
                                            try {
                                                TicketApi.retry(no)
                                                toast(ctx, s(R.string.toast_retry_sent))
                                                nav.requestRefresh()
                                            } catch (_: Exception) { toast(ctx, s(R.string.err_action)) }
                                        }
                                    })
                            }
                        }
                        item { QuadCard(d.optJSONObject("quad")) }
                        item { RiskCard(d.optJSONObject("riskCheck")) }
                        if (status == "DONE") item { ReceiptCard() }
                        item { Spacer(Modifier.height(12.dp)) }
                    }
                }
                BottomActionBar(d, nav, no,
                    onAccept = {
                        scope.launch {
                            try {
                                TicketApi.accept(no)
                                toast(ctx, s(R.string.toast_accepted))
                                nav.pop()
                            } catch (_: Exception) { toast(ctx, s(R.string.err_accept)) }
                        }
                    },
                    onRollback = {
                        scope.launch {
                            try {
                                TicketApi.rollback(no)
                                toast(ctx, s(R.string.toast_rollback_sent))
                                nav.requestRefresh()
                            } catch (_: Exception) { toast(ctx, s(R.string.err_action)) }
                        }
                    },
                    onRetry = {
                        scope.launch {
                            try {
                                TicketApi.retry(no)
                                toast(ctx, s(R.string.toast_retry_sent))
                                nav.requestRefresh()
                            } catch (_: Exception) { toast(ctx, s(R.string.err_action)) }
                        }
                    })
            }
        }
    }
}
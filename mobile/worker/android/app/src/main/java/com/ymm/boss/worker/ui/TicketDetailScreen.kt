package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.ymm.boss.worker.api.TicketApi
import kotlinx.coroutines.launch

/**
 * 工单详情(对齐 designs/worker-order-detail-v1.spec.md):
 * - 4 屏状态分支由 status 字段决定:TODO/DOING/DONE
 * - 类型(INSTALL/REPAIR)由 stages.length 推断(后端 TicketDetail 暂未返 type)
 * - 卡片实现见 TicketDetailCards.kt
 */
@Composable
fun TicketDetailScreen(nav: NavHost, no: String) {
    val state by loadOnce(no) { TicketApi.detail(no) }
    val scope = rememberCoroutineScope()
    val ctx = LocalContext.current

    Column(Modifier.fillMaxSize()) {
        when (val s = state) {
            is Load.Loading -> Box(Modifier.weight(1f)) { Loading() }
            is Load.Fail -> {
                Box(Modifier.weight(1f)) {
                    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
                        TopBar("工单详情", onBack = { nav.pop() }, action = "联系调度",
                            onAction = { nav.push(Screen.Service) })
                        Card(Modifier.padding(14.dp)) {
                            Notice("工单加载失败，请刷新重试。", red = true)
                        }
                    }
                }
            }
            is Load.Ok -> {
                val d = s.data
                val status = d.optString("status")
                val type = inferTicketType(d)
                val title = if (type == "REPAIR") "报障工单" else "工单详情"

                Column(Modifier.weight(1f).verticalScroll(rememberScrollState())) {
                    TopBar(title, onBack = { nav.pop() }, action = "联系调度",
                        onAction = { nav.push(Screen.Service) })
                    DetailHeaderCard(d, type)
                    if (status != "TODO") QuickActionRow(d, nav, no)
                    TimelineCard(d)
                    // A 屏(安装/装维中)追加回退/重试(spec §4.7)
                    if (status == "DOING" && type == "INSTALL") {
                        TimelineExtras(no,
                            onRollback = {
                                scope.launch {
                                    try { toast(ctx, TicketApi.rollback(no)
                                        .optString("message", "已回退上一环节")) }
                                    catch (_: Exception) { toast(ctx, "操作失败，请重试。") }
                                }
                            },
                            onRetry = {
                                scope.launch {
                                    try { toast(ctx, TicketApi.retry(no)
                                        .optString("message", "已重试失败环节")) }
                                    catch (_: Exception) { toast(ctx, "操作失败，请重试。") }
                                }
                            })
                    }
                    QuadCard(d.optJSONObject("quad"))
                    RiskCard(d.optJSONObject("riskCheck"))
                    if (status == "DONE") ReceiptCard()
                    Spacer(Modifier.height(12.dp))
                }
                BottomActionBar(d, nav, no,
                    onAccept = {
                        scope.launch {
                            try { toast(ctx, TicketApi.accept(no)
                                .optString("message", "已领取工单"))
                                nav.pop() }
                            catch (_: Exception) { toast(ctx, "领取失败，请重试。") }
                        }
                    },
                    onRollback = {
                        scope.launch {
                            try { toast(ctx, TicketApi.rollback(no)
                                .optString("message", "已回退")) }
                            catch (_: Exception) { toast(ctx, "操作失败，请重试。") }
                        }
                    },
                    onRetry = {
                        scope.launch {
                            try { toast(ctx, TicketApi.retry(no)
                                .optString("message", "已重试")) }
                            catch (_: Exception) { toast(ctx, "操作失败，请重试。") }
                        }
                    })
            }
        }
    }
}
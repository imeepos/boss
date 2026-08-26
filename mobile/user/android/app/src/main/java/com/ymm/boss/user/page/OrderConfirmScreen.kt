package com.ymm.boss.user.page

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.OrderApi
import com.ymm.boss.user.api.ProductApi
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.EmptyState
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Route
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import com.stripe.android.paymentsheet.PaymentSheet
import com.stripe.android.paymentsheet.PaymentSheetResult
import com.stripe.android.paymentsheet.rememberPaymentSheet
import kotlinx.coroutines.launch
import org.json.JSONObject

/**
 * 订单确认页:套餐信息 + 安装地址选择 + Stripe 卡收款。
 * 进入即加载 product + addresses;默认勾选默认地址(无默认则第一个,无地址则禁用提交并提示去新增)。
 * 提交 = POST /orders 建单(环节1)→ POST /orders/{orderNo}/stripe-intent 拉 PaymentSheet。
 * PaymentSheet 成功 → 跳订单详情;取消/失败 → 留在本页,展示原因。
 */
@Composable
fun OrderConfirmScreen(nav: Nav, productId: String) {
    var product by remember { mutableStateOf<JSONObject?>(null) }
    var addresses by remember { mutableStateOf<List<JSONObject>?>(null) }
    var selectedAddressId by remember { mutableStateOf("") }
    var err by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var orderNo by remember { mutableStateOf("") }
    val paymentSheet = rememberPaymentSheet { result ->
        // PaymentSheet 回调(支付结果异步落这里);成功跳订单详情,失败/取消留本页。
        when (result) {
            is PaymentSheetResult.Completed -> {
                submitting = false
                if (orderNo.isNotBlank()) {
                    nav.resetTo(Route.Order(orderNo))
                } else {
                    nav.resetTo(Route.Orders)
                }
            }
            is PaymentSheetResult.Canceled -> {
                submitting = false
                err = "支付已取消"
            }
            is PaymentSheetResult.Failed -> {
                submitting = false
                err = "支付失败:${result.error.message ?: "请稍后重试"}"
            }
        }
    }

    LaunchedEffect(productId, nav.refreshTick) {
        err = ""
        try {
            val d = ProductApi.detail(productId)
            product = d.optJSONObject("product")
        } catch (e: Exception) { err = "套餐详情加载失败" }
        try {
            val list = ProfileApi.addresses().optJSONArray("items").toObjectList()
            addresses = list
            if (selectedAddressId.isBlank()) {
                selectedAddressId = list.firstOrNull { it.optBoolean("isDefault") }
                    ?.optString("addressId")
                    ?: list.firstOrNull()?.optString("addressId")
                    ?: ""
            }
        } catch (e: Exception) { addresses = emptyList() }
    }

    Box(Modifier.fillMaxSize().background(Palette.bg)) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
            TopBar("确认订单", onBack = { nav.pop() })
            if (err.isNotEmpty()) Notice(err, Palette.err)
            ProductCard(product)
            AddressPickCard(addresses, selectedAddressId) { selectedAddressId = it }
            Notice("提交订单后请尽快完成支付;未完成订单自动取消。")
            Spacer(Modifier.height(96.dp))
        }
        CtaBar(
            product = product,
            enabled = !submitting && product != null && selectedAddressId.isNotBlank(),
            onClick = {
                if (submitting) return@CtaBar
                submitting = true
                err = ""
                scope.launch {
                    try {
                        val orderResp = OrderApi.submit(productId, selectedAddressId)
                        val no = orderResp.optString("orderNo")
                        if (no.isBlank()) throw Api.HttpError(0, "下单失败,缺少订单号")
                        orderNo = no
                        val intent = OrderApi.stripeIntent(no)
                        presentStripeSheet(context, paymentSheet, intent)
                    } catch (e: Exception) {
                        submitting = false
                        err = Api.friendlyMessage(e).ifBlank { "下单失败,请稍后重试" }
                    }
                }
            },
        )
    }
}

@Composable
private fun ProductCard(product: JSONObject?) {
    AppCard {
        CardTitle("套餐信息")
        val name = product?.optString("name")?.takeIf { it.isNotBlank() } ?: "加载中…"
        val fee = product?.optString("monthlyFee")?.takeIf { it.isNotBlank() } ?: "—"
        val desc = product?.optString("description")?.takeIf { it.isNotBlank() } ?: ""
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                name, fontSize = 14.sp, fontWeight = FontWeight.W500, color = Palette.ink,
                modifier = Modifier.weight(1f),
            )
            Text("¥$fee/月", fontSize = 14.sp, color = Palette.primary, fontWeight = FontWeight.W600)
        }
        if (desc.isNotEmpty()) Notice(desc)
    }
}

@Composable
private fun AddressPickCard(
    items: List<JSONObject>?, selected: String, onSelect: (String) -> Unit,
) {
    AppCard {
        CardTitle("选择安装地址", more = if (!items.isNullOrEmpty()) "管理" else null)
        when {
            items == null -> Text("地址加载中…", fontSize = 13.sp, color = Palette.muted)
            items.isEmpty() -> EmptyState("暂无安装地址,请先到「我的-家庭地址管理」新增")
            else -> items.forEach { a ->
                val id = a.optString("addressId")
                AddressRow(a, selected == id) { onSelect(id) }
            }
        }
    }
}

@Composable
private fun AddressRow(a: JSONObject, isSelected: Boolean, onClick: () -> Unit) {
    val isDefault = a.optBoolean("isDefault")
    CellRow(
        title = a.optString("label").ifBlank { "未命名地址" },
        desc = "${a.optString("contact")} · ${a.optString("phoneMasked")}",
        onClick = onClick,
        right = {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Tag(
                    if (isDefault) "默认" else "备用",
                    if (isDefault) Palette.success else Palette.muted,
                )
                Spacer(Modifier.width(6.dp))
                RadioButton(selected = isSelected, onClick = onClick)
            }
        },
    )
}

@Composable
private fun BoxScope.CtaBar(product: JSONObject?, enabled: Boolean, onClick: () -> Unit) {
    val fee = product?.optString("monthlyFee")?.takeIf { it.isNotBlank() } ?: "—"
    Surface(
        tonalElevation = 2.dp,
        color = Color.White,
        modifier = Modifier.fillMaxWidth().align(Alignment.BottomCenter),
    ) {
        Button(
            onClick = onClick,
            enabled = enabled,
            colors = ButtonDefaults.buttonColors(
                containerColor = Palette.primary,
                disabledContainerColor = Palette.muted,
            ),
            shape = RoundedCornerShape(8.dp),
            modifier = Modifier.fillMaxWidth().padding(14.dp).height(44.dp),
        ) {
            Text(
                if (enabled) "确认并支付 ¥$fee/月" else "暂不可提交",
                fontSize = 14.sp, fontWeight = FontWeight.W500, color = Color.White,
            )
        }
    }
}

/** 拉起 Stripe PaymentSheet:clientSecret + merchant 名即可。落账等 /webhooks/stripe 异步完成。 */
private fun presentStripeSheet(
    context: android.content.Context,
    paymentSheet: PaymentSheet,
    intent: JSONObject,
) {
    val clientSecret = intent.optString("clientSecret")
    if (clientSecret.isBlank()) throw Api.HttpError(0, "支付会话创建失败")
    val config = PaymentSheet.Configuration(merchantDisplayName = "BOSS 业务平台")
    paymentSheet.presentWithPaymentIntent(clientSecret, config)
}

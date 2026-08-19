package com.ymm.boss.user.page

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.AppCard
import com.ymm.boss.user.ui.CardTitle
import com.ymm.boss.user.ui.CellRow
import com.ymm.boss.user.ui.FieldLabel
import com.ymm.boss.user.ui.Nav
import com.ymm.boss.user.ui.Notice
import com.ymm.boss.user.ui.Palette
import com.ymm.boss.user.ui.Tag
import com.ymm.boss.user.ui.TopBar
import kotlinx.coroutines.launch
import org.json.JSONObject

// 对应草稿 docs/user/address.html:地址簿,GET/POST /addresses,列表失败回退 /profile.addresses。
@Composable
fun AddressScreen(nav: Nav) {
    var items by remember { mutableStateOf<List<JSONObject>>(emptyList()) }
    LaunchedEffect(Unit) { items = loadAddresses() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
        TopBar("家庭地址管理", { nav.pop() })
        AddressListCard(items)
        AddressFormCard { items = loadAddresses() }
        Notice("地址须挂接到小区/楼栋层级，用于资源核查与装维上门。")
        Spacer(Modifier.height(12.dp))
    }
}

@Composable
private fun AddressListCard(items: List<JSONObject>) {
    AppCard {
        CardTitle("地址列表")
        if (items.isEmpty()) Text("暂无地址", fontSize = 12.5.sp, color = Palette.muted)
        items.forEach { a -> AddressCell(a) }
    }
}

@Composable
private fun AddressFormCard(onSaved: suspend () -> Unit) {
    var msg by remember { mutableStateOf("") }
    var community by remember { mutableStateOf("") }
    var building by remember { mutableStateOf("") }
    var door by remember { mutableStateOf("") }
    var contact by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    AppCard {
        CardTitle("新增地址")
        FieldLabel("小区 / 楼盘")
        AddrInput(community, "请输入小区名") { community = it }
        FieldLabel("楼栋")
        AddrInput(building, "楼栋号") { building = it }
        FieldLabel("门牌号")
        AddrInput(door, "单元/房间号") { door = it }
        FieldLabel("联系人")
        AddrInput(contact, "联系人") { contact = it }
        Notice(msg)
        SaveAddressButton(scope, addressPayload(community, building, door, contact), onSaved) { text ->
            msg = text
        }
    }
}

private suspend fun loadAddresses(): List<JSONObject> =
    try {
        ProfileApi.addresses().optJSONArray("items").toObjectList()
    } catch (e: Exception) {
        try {
            ProfileApi.get().optJSONArray("addresses").toObjectList()
        } catch (e2: Exception) { emptyList() }
    }

private fun addressPayload(community: String, building: String, door: String, contact: String): JSONObject =
    JSONObject()
        .put("community", community)
        .put("building", building)
        .put("door", door)
        .put("contact", contact)

@Composable
private fun SaveAddressButton(
    scope: kotlinx.coroutines.CoroutineScope,
    payload: JSONObject,
    onSaved: suspend () -> Unit,
    onResult: (String) -> Unit,
) {
    Button(
        onClick = {
            scope.launch {
                if (payload.optString("community").isBlank() || payload.optString("building").isBlank() ||
                    payload.optString("door").isBlank()
                ) {
                    onResult("请填写小区、楼栋与门牌号")
                } else {
                    try {
                        ProfileApi.createAddress(payload)
                        onSaved()
                        onResult("已保存")
                    } catch (e: Exception) { onResult("保存失败,请稍后重试") }
                }
            }
        },
        colors = ButtonDefaults.buttonColors(containerColor = Palette.primary),
        modifier = Modifier.fillMaxWidth().height(44.dp).padding(top = 4.dp),
    ) { Text("保存地址") }
}

@Composable
private fun AddrInput(value: String, placeholder: String, onChange: (String) -> Unit) {
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        placeholder = { Text(placeholder, fontSize = 13.sp) },
        singleLine = true,
        modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
    )
}

@Composable
private fun AddressCell(a: JSONObject) {
    val isDefault = a.optBoolean("isDefault")
    CellRow(
        title = a.optString("label"),
        desc = "${if (isDefault) "默认安装地址" else "备用地址"} · ${a.optString("contact")} · ${a.optString("phoneMasked")}",
        right = { Tag(if (isDefault) "默认" else "备用", if (isDefault) Palette.success else Palette.muted) },
    )
}

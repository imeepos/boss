package com.ymm.boss.user.page

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ArrowDropDown
import androidx.compose.material.icons.outlined.Place
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.user.api.Api
import com.ymm.boss.user.api.LocationProvider
import com.ymm.boss.user.api.ProfileApi
import com.ymm.boss.user.api.toObjectList
import com.ymm.boss.user.ui.Palette
import kotlinx.coroutines.CancellationException
import org.json.JSONObject

/**
 * 新增/编辑地址底部弹窗。
 * 顶部"使用当前位置"：GPS 坐标预填门牌号参考（可改写）。
 * "所在区域"行：级联选择地址层级树（大区→市→街道/Barangay），选中回填
 * addressPath(ltree) + 小区名(仅原值为空时覆盖)；未选时仍可手填小区（历史下拉兜底）。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddressEditorSheet(
    initial: JSONObject?,
    recentCommunities: List<String>,
    onDismiss: () -> Unit,
    onSubmit: (JSONObject) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val activity = context as? android.app.Activity

    var community by remember { mutableStateOf(initial?.optString("community").orEmpty()) }
    var building by remember { mutableStateOf(initial?.optString("building").orEmpty()) }
    var door by remember { mutableStateOf(initial?.optString("door").orEmpty()) }
    var contact by remember { mutableStateOf(initial?.optString("contact").orEmpty()) }
    var phone by remember { mutableStateOf("") }
    var addressPath by remember { mutableStateOf(initial?.optString("addressPath").orEmpty()) }
    var regionBreadcrumb by remember { mutableStateOf("") }
    var pickerVisible by remember { mutableStateOf(false) }
    var err by remember { mutableStateOf("") }
    var locationHint by remember { mutableStateOf("") }
    var locating by remember { mutableStateOf(false) }
    var rationaleVisible by remember { mutableStateOf(false) }
    val communityOptions = remember(recentCommunities, community) {
        recentCommunities.filter { it.isNotBlank() && it != community }.distinct()
    }
    // 编辑态回显:addressPath 有值但面包屑空 → /address-tree/lookup 精确反查祖先链回填
    // (search 是模糊 ILIKE+LIMIT,匹配不了 ltree 编码 path,不能反查)。
    // 失败降级:显示 path 末段 code+重试入口;403 服务授权异常必须明示,禁止静默吞掉。
    var lookupFailed by remember { mutableStateOf(false) }
    var lookupForbidden by remember { mutableStateOf(false) }
    var lookupTick by remember { mutableIntStateOf(0) }

    LaunchedEffect(addressPath, lookupTick) {
        val path = addressPath.trim()
        if (path.isBlank() || regionBreadcrumb.isNotBlank()) return@LaunchedEffect
        lookupFailed = false
        lookupForbidden = false
        try {
            val items = ProfileApi.lookupAddressTree(listOf(path)).optJSONArray("items").toObjectList()
            val hit = items.firstOrNull { it.optString("path") == path } ?: items.firstOrNull()
            if (hit == null) {
                lookupFailed = true
                return@LaunchedEffect
            }
            val names = hit.optJSONArray("ancestors").toObjectList().map { it.optString("name") } +
                    hit.optJSONObject("node")?.optString("name").orEmpty()
            val crumb = names.filter { it.isNotBlank() }.joinToString(" · ")
            if (crumb.isBlank()) lookupFailed = true else regionBreadcrumb = crumb
        } catch (e: CancellationException) {
            throw e
        } catch (e: Api.HttpError) {
            // HTTP 403 与信封 40300 都按服务授权异常处理(102 实测 403 LICENSE_REQUIRED)
            if (e.status == 403 || e.status == 40300) lookupForbidden = true else lookupFailed = true
        } catch (e: Exception) {
            lookupFailed = true
        }
    }

    fun launchLocation() = doLocate(
        context, scope,
        setLocating = { locating = it },
        setHint = { locationHint = it },
        setDoor = { door = it },
        setErr = { err = it },
    )

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions(),
    ) { granted ->
        val ok = granted[Manifest.permission.ACCESS_FINE_LOCATION] == true ||
                granted[Manifest.permission.ACCESS_COARSE_LOCATION] == true
        if (ok) launchLocation() else { err = "未授予定位权限"; locationHint = "" }
    }

    fun startPermissionFlow() {
        // Activity 上报：true 表示用户拒绝过且未勾"不再询问"；此时直接再 launch
        // 系统不会再弹窗，必须先 rationale。
        if (activity != null && activity.shouldShowRequestPermissionRationale(
                Manifest.permission.ACCESS_FINE_LOCATION)) {
            rationaleVisible = true
        } else {
            permissionLauncher.launch(arrayOf(
                Manifest.permission.ACCESS_FINE_LOCATION,
                Manifest.permission.ACCESS_COARSE_LOCATION,
            ))
        }
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = Palette.panel,
    ) {
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)
                .verticalScroll(rememberScrollState()),
        ) {
            SheetHeader(title = if (initial == null) "新增家庭地址" else "编辑家庭地址", onClose = onDismiss)
            Spacer(Modifier.height(4.dp))
            LocateAction(
                locating = locating,
                hint = locationHint,
                onClick = {
                    err = ""
                    if (LocationProvider.hasPermission(context)) launchLocation()
                    else startPermissionFlow()
                },
            )
            Spacer(Modifier.height(8.dp))
            FieldLabel("所在区域")
            RegionRow(
                breadcrumb = regionBreadcrumb,
                addressPath = addressPath,
                lookupFailed = lookupFailed,
                lookupForbidden = lookupForbidden,
                onRetry = { lookupTick++ },
                onClick = { pickerVisible = true },
            )
            FieldLabel("小区 / 街道名")
            CommunityField(community, communityOptions) { community = it; err = "" }
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Box(Modifier.weight(1f)) {
                    FieldLabel("楼栋")
                    AddrInput(building, "楼栋号", KeyboardType.Text) { building = it }
                }
                Box(Modifier.weight(1.4f)) {
                    FieldLabel("门牌号")
                    AddrInput(door, "单元/房间号", KeyboardType.Text) { door = it }
                }
            }
            FieldLabel("联系人")
            AddrInput(contact, "联系人姓名", KeyboardType.Text) { contact = it; err = "" }
            FieldLabel("联系电话")
            AddrInput(phone, "留空则使用账户手机号", KeyboardType.Phone) { phone = it; err = "" }

            if (err.isNotBlank()) Text(err, fontSize = 12.sp, color = Palette.err,
                modifier = Modifier.padding(top = 4.dp))

            Spacer(Modifier.height(12.dp))
            SubmitButton(
                text = if (initial == null) "保存地址" else "保存修改",
                enabled = community.isNotBlank() && contact.isNotBlank() && phoneOk(phone),
            ) {
                val payload = JSONObject()
                    .put("community", community.trim())
                    .put("building", building.trim())
                    .put("door", door.trim())
                    .put("contact", contact.trim())
                    .put("phone", phone.trim())
                    .put("addressPath", addressPath.trim())
                onSubmit(payload)
            }
            Spacer(Modifier.height(8.dp))
            TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) { Text("取消") }
            Spacer(Modifier.height(8.dp))
        }
    }

    if (pickerVisible) {
        AddressRegionPickerSheet(
            onDismiss = { pickerVisible = false },
            onSelected = { sel ->
                pickerVisible = false
                addressPath = sel.addressPath
                regionBreadcrumb = sel.breadcrumb
                // 仅原值为空才回填:用户手填的小区名(历史下拉兜底)不被选区覆盖。
                if (community.isBlank()) community = sel.community
                err = ""
            },
        )
    }

    if (rationaleVisible) {
        AlertDialog(
            onDismissRequest = { rationaleVisible = false },
            title = { Text("需要定位权限") },
            text = { Text("获取当前位置用于把经纬度填入门牌号参考，地址仍可手动修改。") },
            confirmButton = {
                TextButton(onClick = {
                    rationaleVisible = false
                    permissionLauncher.launch(arrayOf(
                        Manifest.permission.ACCESS_FINE_LOCATION,
                        Manifest.permission.ACCESS_COARSE_LOCATION,
                    ))
                }) { Text("继续") }
            },
            dismissButton = {
                TextButton(onClick = { rationaleVisible = false }) { Text("暂不开启") }
            },
        )
    }
}


package com.ymm.boss.worker.ui

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.ymm.boss.worker.ui.theme.Ink

// 表单域标签:输入卡片(OutlinedTextField)上方的 13sp 中灰标签,与 KvRow 同风格。
// 由 Charge/Complaint/Replace/Reschedule 等表单页共用。
@Composable
fun FieldLabel(text: String) {
    Text(
        text,
        modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp),
        fontSize = 13.sp,
        fontWeight = FontWeight.Medium,
        color = Ink,
    )
}

package com.ymm.boss.user.page

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 实名上传压缩策略单测(inSampleSize 计算):
 * 目标是 "把 >2MB 拍摄图压到 1MB 以内且仍可读",样本必须保清晰又能降内存。
 * 这是 fix(realname-upload-413) 的回归门禁:42200/413 上传错误往往由
 * inSampleSize 计算错误导致图片超过反代限制。
 */
class RealNameUploadTest {

    @Test
    fun `inSampleSize returns 1 for already small image`() {
        // 1000x1000 < 1600*2,样本保持 1
        assertEquals(1, computeInSampleSize(1000, 1000, 1600))
    }

    @Test
    fun `inSampleSize doubles for huge image`() {
        // 8000x6000 长边 8000 > 1600*2=3200 → 2;8000/2=4000 > 3200 → 4
        assertEquals(4, computeInSampleSize(8000, 6000, 1600))
    }

    @Test
    fun `inSampleSize respects power-of-two`() {
        // 4032x3024(常见手机最大尺寸) → 2
        // longEdge=4032, 4032/1=4032 > 3200 → 2;4032/2=2016 ≤ 3200 → 2
        assertEquals(2, computeInSampleSize(4032, 3024, 1600))
    }

    @Test
    fun `inSampleSize caps at extreme resolution`() {
        // 24000x6000 长边 24000 → 1,2,4,8(24000/8=3000≤3200) → 8
        assertEquals(8, computeInSampleSize(24000, 6000, 1600))
    }

    @Test
    fun `inSampleSize is always power of two`() {
        for (w in listOf(800, 2000, 4032, 6000, 12000)) {
            for (h in listOf(600, 1500, 3024, 4000, 9000)) {
                val s = computeInSampleSize(w, h, 1600)
                assertTrue("w=$w h=$h sample=$s", s > 0 && (s and (s - 1)) == 0)
            }
        }
    }
}

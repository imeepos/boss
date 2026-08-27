package com.ymm.boss.user.page

import org.junit.Assert.assertEquals
import org.junit.Test

class OrderTimelineLogicTest {

    @Test
    fun `currentMilestoneOf maps 12 stages to spec milestones`() {
        // 设计稿映射(contract/terms.md §1 12 环节):stage≤1→1 / 2-7→2 / 8-11→3 / 12→4。
        // 曾按 3 个一组分桶,环节 10-11(激活中)误落「完成」组,这里逐 stage 断言防回归。
        val expect = mapOf(
            1 to 1, 2 to 2, 3 to 2, 4 to 2, 5 to 2, 6 to 2, 7 to 2,
            8 to 3, 9 to 3, 10 to 3, 11 to 3, 12 to 4,
        )
        expect.forEach { (stage, milestone) ->
            assertEquals("stage=$stage", milestone, currentMilestoneOf(listOf(stage)))
        }
        // 组内最大 stage 定组:group 2 含 2-7,group 3 含 8-11
        assertEquals(2, currentMilestoneOf(listOf(2, 7)))
        assertEquals(3, currentMilestoneOf(listOf(8, 11)))
    }

    @Test
    fun `currentMilestoneOf empty timeline defaults to milestone 1`() {
        assertEquals(1, currentMilestoneOf(emptyList()))
    }

    @Test
    fun `currentMilestoneOf clamps out-of-range stages`() {
        assertEquals(4, currentMilestoneOf(listOf(99)))
        assertEquals(1, currentMilestoneOf(listOf(0)))
    }

    @Test
    fun `milestoneStateOf all done is DONE`() {
        assertEquals(MilestoneState.DONE, milestoneStateOf(listOf("DONE", "DONE", "DONE")))
    }

    @Test
    fun `milestoneStateOf any done or doing is DOING`() {
        assertEquals(MilestoneState.DOING, milestoneStateOf(listOf("DONE", "DOING", "PENDING")))
        assertEquals(MilestoneState.DOING, milestoneStateOf(listOf("DONE", "PENDING")))
    }

    @Test
    fun `milestoneStateOf all pending or empty is PENDING`() {
        assertEquals(MilestoneState.PENDING, milestoneStateOf(listOf("PENDING", "PENDING")))
        assertEquals(MilestoneState.PENDING, milestoneStateOf(emptyList()))
    }

    @Test
    fun `cancelStageDesc covers stage boundaries`() {
        assertEquals("订单创建后立即撤销,通常为重复下单或暂不需要", cancelStageDesc(1))
        assertEquals("资源核查/端口预占阶段被取消,该地址可能暂无可用资源", cancelStageDesc(3))
        assertEquals("合同收费环节被取消,可能为支付未完成或主动撤销", cancelStageDesc(6))
        assertEquals("已派单后被取消,可能为装维条件不具备或用户主动撤销", cancelStageDesc(9))
        assertEquals("上门安装过程中被取消,可能为现场条件不满足", cancelStageDesc(11))
        assertEquals("订单完成后异常取消,请联系客服核查", cancelStageDesc(12))
    }
}

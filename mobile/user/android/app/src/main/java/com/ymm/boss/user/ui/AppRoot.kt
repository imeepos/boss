package com.ymm.boss.user.ui

import androidx.compose.runtime.Composable
import com.ymm.boss.user.page.AddonScreen
import com.ymm.boss.user.page.AddressScreen
import com.ymm.boss.user.page.AgreementScreen
import com.ymm.boss.user.page.BillScreen
import com.ymm.boss.user.page.BillsScreen
import com.ymm.boss.user.page.CancelScreen
import com.ymm.boss.user.page.ChangeScreen
import com.ymm.boss.user.page.ComplaintScreen
import com.ymm.boss.user.page.CouponScreen
import com.ymm.boss.user.page.DiyScreen
import com.ymm.boss.user.page.FaultDetailScreen
import com.ymm.boss.user.page.FaultScreen
import com.ymm.boss.user.page.ForgotScreen
import com.ymm.boss.user.page.HelpScreen
import com.ymm.boss.user.page.HomeScreen
import com.ymm.boss.user.page.InvoiceScreen
import com.ymm.boss.user.page.LoginScreen
import com.ymm.boss.user.page.MessagesScreen
import com.ymm.boss.user.page.MoveScreen
import com.ymm.boss.user.page.MyPlanScreen
import com.ymm.boss.user.page.NotifyScreen
import com.ymm.boss.user.page.OrderScreen
import com.ymm.boss.user.page.OrdersScreen
import com.ymm.boss.user.page.PayResultScreen
import com.ymm.boss.user.page.PayScreen
import com.ymm.boss.user.page.ProductScreen
import com.ymm.boss.user.page.ProductsScreen
import com.ymm.boss.user.page.ProfileScreen
import com.ymm.boss.user.page.RateScreen
import com.ymm.boss.user.page.ReceiptScreen
import com.ymm.boss.user.page.RegisterScreen
import com.ymm.boss.user.page.SecurityScreen
import com.ymm.boss.user.page.ServiceScreen
import com.ymm.boss.user.page.TopupScreen
import com.ymm.boss.user.page.UsageScreen
import com.ymm.boss.user.page.VerifyScreen

/** 路由 → 页面组合映射;每个 Screen 签名统一 (nav: Nav, [参数])。 */
@Composable
fun RouteScreen(route: Route, nav: Nav) {
    when (route) {
        Route.Login -> LoginScreen(nav)
        Route.Register -> RegisterScreen(nav)
        Route.Forgot -> ForgotScreen(nav)
        Route.Verify -> VerifyScreen(nav)
        Route.Agreement -> AgreementScreen(nav)
        Route.Home -> HomeScreen(nav)
        Route.Products -> ProductsScreen(nav)
        is Route.Product -> ProductScreen(nav, route.id)
        Route.Orders -> OrdersScreen(nav)
        is Route.Order -> OrderScreen(nav, route.no)
        is Route.Rate -> RateScreen(nav, route.no)
        Route.MyPlan -> MyPlanScreen(nav)
        is Route.Change -> ChangeScreen(nav, route.planId)
        is Route.Cancel -> CancelScreen(nav, route.planId)
        is Route.Move -> MoveScreen(nav, route.planId)
        Route.Bills -> BillsScreen(nav)
        is Route.Bill -> BillScreen(nav, route.no)
        Route.Pay -> PayScreen(nav)
        Route.PayResult -> PayResultScreen(nav)
        Route.Topup -> TopupScreen(nav)
        Route.Invoice -> InvoiceScreen(nav)
        Route.Coupon -> CouponScreen(nav)
        is Route.Receipt -> ReceiptScreen(nav, route.payNo)
        Route.Fault -> FaultScreen(nav)
        is Route.FaultDetail -> FaultDetailScreen(nav, route.no)
        Route.Complaint -> ComplaintScreen(nav)
        Route.Service -> ServiceScreen(nav)
        Route.Diy -> DiyScreen(nav)
        Route.Help -> HelpScreen(nav)
        Route.Messages -> MessagesScreen(nav)
        Route.Notify -> NotifyScreen(nav)
        Route.Profile -> ProfileScreen(nav)
        Route.Security -> SecurityScreen(nav)
        Route.Usage -> UsageScreen(nav)
        Route.Address -> AddressScreen(nav)
        Route.Addon -> AddonScreen(nav)
    }
}

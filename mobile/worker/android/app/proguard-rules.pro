-keep class com.ymm.boss.worker.** { *; }
-keep class androidx.camera.** { *; }
-keep class com.google.mlkit.** { *; }
-keep class com.google.android.gms.vision.** { *; }
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
-assumenosideeffects class android.util.Log {
    public static boolean isLoggable(java.lang.String, int);
    public static int v(...);
    public static int d(...);
    public static int i(...);
}


# 极光推送(官方混淆要求;jcore/jpush 反射回调不可裁剪)
-keep class cn.jpush.** { *; }
-keep class cn.jiguang.** { *; }
-dontwarn cn.jpush.**
-dontwarn cn.jiguang.**

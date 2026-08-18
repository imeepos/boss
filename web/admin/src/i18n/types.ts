// i18n 翻译 key 类型定义:所有页面/组件文本集中管理。

export type Locale = 'zh-CN' | 'en-US' | 'ms-MY'

export interface Translations {
  common: {
    brand: string
    tagline: string
    footer: string
    logout: string
    loading: string
    dataScope: string
    dataScopeAll: string
  }
  auth: {
    login: {
      title: string
      subtitle: string
      usernamePlaceholder: string
      passwordPlaceholder: string
      submit: string
      submitting: string
      noAccount: string
      toRegister: string
      emptyFields: string
      fail: string
    }
    register: {
      title: string
      subtitle: string
      namePlaceholder: string
      usernamePlaceholder: string
      passwordPlaceholder: string
      confirmPlaceholder: string
      submit: string
      submitting: string
      hasAccount: string
      toLogin: string
      emptyName: string
      invalidUsername: string
      shortPassword: string
      mismatch: string
      fail: string
      tip: string
    }
  }
  ads: Array<{ title: string; sub: string }>
  shell: {
    home: string
    searchPlaceholder: string
    notifications: string
    themeToDark: string
    themeToLight: string
    collapseMenu: string
    expandMenu: string
    quickCreate: string
    version: string
  }
  menu: {
    groups: Record<string, string>
    items: Record<string, string>
  }
  pages: {
    dashboard: {
      title: string
      welcome: string
    }
    placeholder: {
      building: string
    }
    account: {
      title: string
      searchPlaceholder: string
      columns: string[]
      edit: string
      allGroup: string
      total: string
      loadFail: string
    }
    error: {
      forbidden: string
      notFound: string
    }
  }
}
// i18n 翻译 key 类型定义:所有页面/组件文本集中管理。

export type Locale = 'zh-CN' | 'en-US' | 'ms-MY'

export interface Translations {
  common: {
    brand: string
    tagline: string
    footer: string
    logout: string
    profile: string
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
    geo: {
      title: string
      tabCountry: string
      tabSubdiv: string
      countryColumns: string[]
      subdivColumns: string[]
      add: string
      edit: string
      save: string
      cancel: string
      disable: string
      enable: string
      detail: string
      active: string
      inactive: string
      total: string
      loadFail: string
      saveFail: string
      filterCountry: string
      searchPlaceholder: string
      empty: string
      prev: string
      next: string
      perPage: string
      rangeText: string
      jumpText: string
      pageUnit: string
      names: string
      addName: string
      attrs: string
      timeZones: string
      currencies: string
      callingCodes: string
      countryFields: {
        alpha2: string
        alpha3: string
        numericCode: string
        shortName: string
        fullName: string
        m49Region: string
        postalRegex: string
        continent: string
        status: string
      }
      subdivFields: {
        code: string
        countryCode: string
        parentCode: string
        category: string
        level: string
        osmAdminLevel: string
        geonameId: string
      }
    }
    address: {
      title: string
      expand: string
      collapse: string
      attach: string
      unlinked: string
      unlinkedAll: string
      searchAll: string
      noHit: string
      addRoot: string
      addChild: string
      rename: string
      delete: string
      deleteConfirm: string
      deleteFail: string
      pathLabel: string
      pathHint: string
      nameLabel: string
      country: string
      adminCode: string
      none: string
      loadFail: string
      saveFail: string
      empty: string
    }
    importer: {
      title: string
      addrTitle: string
      addrHint: string
      geoTitle: string
      geoHint: string
      importBtn: string
      parseFail: string
      imported: string
      loadFail: string
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
    profile: {
      title: string
      activeAccount: string
      save: string
      cancel: string
      navigation: { title: string; overview: string; overviewDesc: string; personal: string; personalDesc: string; security: string; securityDesc: string; apiKey: string; apiKeyDesc: string; myData: string; myDataDesc: string; backAdmin: string; platformWorkspace: string; menu: string }
      overview: { title: string; desc: string; accountStatus: string; accountStatusDesc: string; quickAccess: string; dispatch: string; dispatchDesc: string; order: string; orderDesc: string; complaint: string; complaintDesc: string; permissionSummary: string; dataScopeSummary: string }
      personal: { title: string; desc: string; badge: string; username: string; realName: string; phone: string; phonePlaceholder: string; email: string; emailPlaceholder: string; saved: string; role: string; company: string; dataScope: string; unassigned: string; allScope: string }
      password: { title: string; desc: string; old: string; next: string; confirm: string; submit: string; passwordPending: string }
      securityProtection: { title: string; loginProtection: string; loginHistory: string; pending: string }
      apiKey: { title: string; desc: string; placeholder: string; name: string; key: string; lastUsed: string; status: string; namePlaceholder: string; create: string; neverUsed: string; active: string; revoke: string; securityTip: string }
      myData: { title: string; desc: string; orders: string; ordersDesc: string; bills: string; billsDesc: string; service: string; serviceDesc: string; messages: string; messagesDesc: string; audit: string; auditDesc: string; permissions: string; permissionsDesc: string }
    }
    error: {
      forbidden: string
      notFound: string
    }
  }
}
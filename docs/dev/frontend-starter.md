本项目使用vite脚手架生成，生成命令如下：
```
npm create vue@latest frontend -- --typescript --router --eslint --prettier
```

技术要求：
- 前端框架：Vue3
- 构建工具：Vite
- 语言：TypeScript
- UI 组件库：Element Plus
- 状态管理：不使用第三方状态管理库，使用 Vue3 自带的组合式 API（Composition API）进行状态管理
- 路由管理：Vue Router
- HTTP 客户端：Axios
- 样式处理：使用 SCSS 预处理器
- 代码质量工具：ESLint（代码风格检查），Prettier（代码格式化）
- 包管理：npm

项目结构：
```
frontend/
├── public/                      # 静态资源目录
│   ├── favicon.ico             # 网站图标
│   └── images/                 # 公共图片资源
└── src/
    ├── main.ts                 # 应用入口文件
    ├── App.vue                 # 根组件
    ├── assets/                 # 资源文件目录
    │   ├── styles/            # 样式文件
    │   │   └──  main.scss      # 全局样式文件
    │   └── images/            # 图片资源
    ├── components/             # 公共组件目录
    ├── views/                  # 页面视图目录
    │   └── home/              # 页面目录
    │       └── index.vue      # 页面组件
    ├── router/                 # 路由配置
    │   └── index.ts           # 路由定义
    ├── composables/            # 组合式函数目录
    │   └── useExample.ts      # 可复用的组合式逻辑
    ├── api/                    # API接口目录
    │   ├── user.ts           # 对应模块接口
    ├── types/                  # TypeScript类型定义
    │   └── index.ts
    ├── utils/                  # 工具函数目录
    │   └── helpers.ts
    ├── constants/              # 常量定义
    │   └── index.ts
    └── config/                 # 配置文件
        └── index.ts
```

开发规范：

1. **命名规范**
   - 组件文件：PascalCase（如 `UserProfile.vue`）
   - 组合式函数：use前缀 + camelCase（如 `useUserData.ts`）
   - 工具函数：camelCase（如 `formatDate.ts`）
   - 常量：UPPER_SNAKE_CASE（如 `API_BASE_URL`）

2. **代码风格**
   - 使用ESLint进行代码检查
   - 使用Prettier进行代码格式化
   - 组件内使用 `<script setup>` 语法
   - 和页面强关联的组件放在页面同目录下，公共组件放在 `components/` 目录下
   - 每个vue和ts文件最好不要超过300行
   - 减少CSS的使用，优先服用组件内置CSS和布局方案

3. **TypeScript使用**
   - 所有文件使用 `.ts` 或 `.vue` 扩展名
   - 为函数参数和返回值添加类型注解
   - 使用interface定义数据结构
   - 避免使用 `any` 类型

如果是第一次生成项目，使用脚手架创建项目后，仅整理/生成目录结构（不包括文件），不新增示例代码，生成后请尝试运行，解决编译错误。
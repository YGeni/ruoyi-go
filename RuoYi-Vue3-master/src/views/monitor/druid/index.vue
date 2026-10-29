<template>
   <div>
      <i-frame v-model:src="url"></i-frame>
   </div>
</template>

<script setup>
import iFrame from '@/components/iFrame'

import { ref } from 'vue'
import { getToken } from '@/utils/auth'

// 后端这个页面要校验登录态，但 iframe 的 src 由浏览器直接发 GET，
// **带不上 Authorization 头**，所以只能把 token 拼在 query 上
// （后端 /druid/login.html 会做 header 优先、query 兜底）。
const url = ref(import.meta.env.VITE_APP_BASE_API + '/druid/login.html?token=' + getToken())
</script>

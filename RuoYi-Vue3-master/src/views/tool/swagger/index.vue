<template>
   <i-frame v-model:src="url"></i-frame>
</template>

<script setup>
import iFrame from '@/components/iFrame'
import { getToken } from '@/utils/auth'

// iframe 的 src 由浏览器直接发 GET，带不上 Authorization 头，
// 所以把 token 拼在 query 上（后端 /swagger-ui/index.html 做 header 优先、query 兜底）。
const url = ref(import.meta.env.VITE_APP_BASE_API + "/swagger-ui/index.html?token=" + getToken())
</script>

import compression from 'vite-plugin-compression'

export default function createCompression(env) {
  const { VITE_BUILD_COMPRESS } = env
  const plugin = []
  if (VITE_BUILD_COMPRESS) {
    const compressList = VITE_BUILD_COMPRESS.split(',')
    if (compressList.includes('gzip')) {
      // gzip 产物需由 Nginx 等静态服务器配置对应的解压缩/回源规则
      plugin.push(
        compression({
          ext: '.gz',
          deleteOriginFile: false
        })
      )
    }
    if (compressList.includes('brotli')) {
      plugin.push(
        compression({
          ext: '.br',
          algorithm: 'brotliCompress',
          deleteOriginFile: false
        })
      )
    }
  }
  return plugin
}

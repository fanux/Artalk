import type { ListFetchParams, ArtalkPlugin } from '@/types'

export const Fetch: ArtalkPlugin = (ctx) => {
  ctx.on('list-fetch', (_params) => {
    if (ctx.getData().getLoading()) return
    ctx.getData().setLoading(true)

    const params: ListFetchParams = {
      // default params
      offset: 0,
      limit: ctx.conf.pagination.pageSize,
      flatMode: ctx.conf.flatMode as boolean, // always be boolean because had been handled in Artalk.init
      paramsModifier: ctx.conf.listFetchParamsModifier,
      ..._params
    }

    // must before other function call
    ctx.getData().setListLastFetch({
      params
    })

    ctx.getApi().comment
      .get(params.offset, params.limit, params.flatMode, params.paramsModifier)
      .then((data) => {
        // Must before all other function call and event trigger,
        // because it will depend on the lastData
        // TODO: this is global variable, easy to use, but not good, consider to refactor.
        // refactor work is hard, because it is used in many places.
        ctx.getData().setListLastFetch({ params, data })

        // 装置评论
        ctx.getData().loadComments(data.comments)

        // 更新页面数据
        ctx.getData().updatePage(data.page)

        // 未读消息提示功能
        ctx.getData().updateUnreads(data.unread || [])

        // trigger events when success
        params.onSuccess && params.onSuccess(data)

        ctx.trigger('list-fetched', { params, data })
      })
      .catch((e) => {
        // 显示错误对话框
        const error = {
          msg: e.msg || String(e),
          data: e.data
        }

        params.onError && params.onError(error)

        // trigger events when error
        ctx.trigger('list-error', error)
        ctx.trigger('list-fetched', { params, error })

        throw e
      })
      .finally(() => {
        ctx.getData().setLoading(false)
      })
  })
}

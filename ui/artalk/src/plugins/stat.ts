import type { ContextApi, ArtalkPlugin } from '@/types'
import Api from '../api'

export interface CountOptions {
  getApi(): Api

  pageKey: string
  countEl: string
  pvEl: string

  /** 是否增加当前页面 PV 数 */
  pvAdd?: boolean
}

export const PvCountWidget: ArtalkPlugin = (ctx: ContextApi) => {
  ctx.on('conf-loaded', () => {
    initCountWidget({
      getApi: () => ctx.getApi(),
      pageKey: ctx.conf.pageKey,
      countEl: ctx.conf.countEl,
      pvEl: ctx.conf.pvEl,
      pvAdd: true,
    })
  })
}

/** 初始化评论数和 PV 数量展示元素 */
export async function initCountWidget(opt: CountOptions) {
  // 评论数
  if (opt.countEl && document.querySelector(opt.countEl)) {
    refreshStatCount(opt, { query: 'page_comment', numEl: opt.countEl })
  }

  // PV
  const initialData = opt.pvAdd ? {
    [opt.pageKey]: (await opt.getApi().page.pv()) // pv+1 and get pv count
  } : undefined

  if (opt.pvEl && document.querySelector(opt.pvEl)) {
    refreshStatCount(opt, {
      query: 'page_pv',
      numEl: opt.pvEl,
      data: initialData,
    })
  }
}

type CountData = { [pageKey: string]: number }

async function refreshStatCount(
  opt: CountOptions,
  args: {
    query: 'page_pv' | 'page_comment'
    numEl: string
    data?: CountData
  }
) {
  let data: CountData = args.data || {}

  // Get page keys which will be queried
  let queryPageKeys = Array.from(document.querySelectorAll(args.numEl))
    .map((e) => e.getAttribute('data-page-key') || opt.pageKey)
    .filter((k) => typeof data[k] !== 'number') // filter out keys that already have data

  queryPageKeys = [...new Set(queryPageKeys)] // deduplicate

  // Fetch count data from server
  if (queryPageKeys.length > 0) {
    const res = await opt.getApi().page.stat(args.query, queryPageKeys) as CountData
    data = { ...data, ...res }
  }

  applyCountData(args.numEl, data, data[opt.pageKey])
}

function applyCountData(selector: string, data: CountData, defaultCount: number) {
  document.querySelectorAll(selector).forEach((el) => {
    const pageKey = el.getAttribute('data-page-key')
    const count = Number(pageKey ? (data[pageKey]) : defaultCount)
    el.innerHTML = `${count}`
  })
}

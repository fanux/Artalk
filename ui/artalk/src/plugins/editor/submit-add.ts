import type { CommentData } from '@/types'
import type PlugKit from './_kit'

export default class SubmitAddPreset {
  constructor(private kit: PlugKit) {}

  async reqAdd() {
    const nComment = await this.kit.useApi().comment.add({
      ...this.getSubmitAddParams()
    })
    return nComment
  }

  getSubmitAddParams() {
    const { nick, email, link } = this.kit.useUser().getData()
    const conf = this.kit.useConf()

    return {
      content: this.kit.useEditor().getContentFinal(),
      nick, email, link,
      rid: 0,
      page_key: conf.pageKey,
      page_title: conf.pageTitle,
      site_name: conf.site
    }
  }

  postSubmitAdd(commentNew: CommentData) {
    // insert the new comment to list
    this.kit.useGlobalCtx().getData().insertComment(commentNew)
  }
}

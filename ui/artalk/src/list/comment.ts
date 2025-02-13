import type { ContextApi, CommentData } from '@/types'
import Comment from '../comment/comment'

export function createComment(ctx: ContextApi, comment: CommentData, ctxComments: CommentData[]): Comment {
  const instance = new Comment(ctx, comment, {
    isFlatMode: ctx.getData().getListLastFetch()?.params.flatMode!,
    afterRender: () => {
      ctx.trigger('comment-rendered', instance)
    },
    onDelete: (c: Comment) => {
      ctx.getData().deleteComment(c.getID())
    },
    replyTo: (comment.rid ? ctxComments.find(c => c.id === comment.rid) : undefined)
  })

  // 渲染元素
  instance.render()

  return instance
}

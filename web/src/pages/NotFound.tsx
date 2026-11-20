import { Link } from 'react-router-dom';
import { EmptyState } from '../components/EmptyState';

export function NotFoundPage() {
  return (
    <main className="wrap">
      <EmptyState
        title="页面不存在"
        desc="链接可能已失效，或者这个地址还没有内容。可以回到榜单，或搜索站点与模型。"
        action="回到榜单"
        actionTo="/"
      />
      <p className="fine" style={{ textAlign: 'center', marginTop: 4 }}>
        也可以看 <Link to="/stations">站点目录</Link> · <Link to="/models">模型价格</Link> · <Link to="/faq">常见问题</Link>
      </p>
    </main>
  );
}

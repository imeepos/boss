// 版本发布页接口层:re-export api/clientReleases,页面只从一个入口引。
export {
  listReleases, patchRelease, uploadRelease,
  type ClientReleaseDTO, type ReleasePatchInput, type ReleaseUploadInput,
} from '../../../api/clientReleases'

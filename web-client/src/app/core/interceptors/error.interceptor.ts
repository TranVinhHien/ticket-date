import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { catchError, throwError } from 'rxjs';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      let errorMessage = 'Đã có lỗi xảy ra. Vui lòng thử lại sau.';

      if (error.status === 0) {
        errorMessage = 'Không thể kết nối đến máy chủ hoặc bị chặn bởi CORS (Failed to fetch). Vui lòng kiểm tra Gateway Backend.';
      } else if (error.error?.message) {
        errorMessage = error.error.message;
      } else if (error.status === 404) {
        errorMessage = 'Không tìm thấy dữ liệu yêu cầu.';
      } else if (error.status === 409) {
        errorMessage = 'Dữ liệu bị trùng lặp hoặc vé không còn đủ số lượng.';
      } else if (error.status >= 500) {
        errorMessage = 'Lỗi hệ thống máy chủ. Vui lòng thử lại sau.';
      }

      return throwError(() => ({
        status: error.status,
        message: errorMessage,
        raw: error
      }));
    })
  );
};

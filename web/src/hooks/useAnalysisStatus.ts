// Analysis status hook
import { useState, useEffect, useCallback } from 'react';
import axios from 'axios';

interface AnalysisStatus {
    id: string;
    status: 'pending' | 'processing' | 'completed' | 'failed';
    progress: number;
    message?: string;
    resultsId?: string;
}

export function useAnalysisStatus(analysisId: string | null) {
    const [status, setStatus] = useState<AnalysisStatus | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isPolling, setIsPolling] = useState(true);

    const fetchStatus = useCallback(async () => {
        if (!analysisId) return;

        try {
            const response = await axios.get(`/api/analyses/${analysisId}/status`);
            setStatus(response.data);

            // Stop polling if completed or failed
            if (response.data.status === 'completed' || response.data.status === 'failed') {
                setIsPolling(false);
            }
        } catch (err: any) {
            console.error('Failed to fetch status:', err);
            // Use backend-provided error messages when available
            if (err.response?.data?.message) {
                setError(err.response.data.message);
            } else if (err.response?.status === 413) {
                setError('Recording file is too large for upload.');
            } else if (!navigator.onLine) {
                setError('No internet connection. Please check your network.');
            } else {
                setError('Failed to fetch analysis status');
            }
            setIsPolling(false);
        }
    }, [analysisId]);

    useEffect(() => {
        if (!analysisId || !isPolling) return;

        // Initial fetch
        fetchStatus();

        // Poll every 2 seconds
        const interval = setInterval(fetchStatus, 2000);

        return () => clearInterval(interval);
    }, [analysisId, isPolling, fetchStatus]);

    return {
        status,
        error,
        isComplete: status?.status === 'completed',
        isFailed: status?.status === 'failed',
        progress: status?.progress || 0,
        message: status?.message
    };
}


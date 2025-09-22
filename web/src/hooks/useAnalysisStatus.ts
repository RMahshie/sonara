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
        } catch (err) {
            console.error('Failed to fetch status:', err);
            setError('Failed to fetch analysis status');
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

